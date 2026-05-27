package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"learncode/internal/executor"
	"learncode/internal/llm"
	"learncode/internal/model"
	"learncode/internal/scraper"
)

// DiscoveredVersion is a single version discovered for a programming language.
type DiscoveredVersion struct {
	Version     string   `json:"version"`
	LTS         bool     `json:"lts"`
	Released    string   `json:"released"`
	Brief       string   `json:"brief"`
	DownloadURL string   `json:"download_url,omitempty"`
	ImageTag    string   `json:"image_tag,omitempty"`
	Source      string   `json:"source,omitempty"`
	DockerRefs  []string `json:"docker_refs,omitempty"`
}

type InitSuggestion struct {
	Name               string              `json:"name"`
	Slug               string              `json:"slug"`
	Icon               string              `json:"icon"`
	CompatibilityModel string              `json:"compatibility_model"`
	Description        string              `json:"description"`
	Versions           []DiscoveredVersion `json:"versions"`
	LatestVersion      string              `json:"latest_version"`
}

type InitConfirmInput struct {
	Name               string              `json:"name"`
	Slug               string              `json:"slug"`
	Icon               string              `json:"icon"`
	CompatibilityModel string              `json:"compatibility_model"`
	DocsURL            string              `json:"docs_url,omitempty"`
	RuntimeURL         string              `json:"runtime_url,omitempty"`
	Versions           []string            `json:"versions"`
	DiscoveredVersions []DiscoveredVersion `json:"discovered_versions,omitempty"`
}

type InitResult struct {
	Language            *model.Language         `json:"language"`
	Versions            []model.LanguageVersion `json:"versions"`
	InitializedVersions []string                `json:"initialized_versions"`
}

type LanguageInitService struct {
	LangSvc        *LanguageService
	VersionSvc     *VersionService
	LLM            *llm.Service
	PromptDir      string
	Scraper        *scraper.Client
	InitVersionSvc *InitService
	KBBuildSvc     *KBBuildService
}

// ─── Internal LLM response types ──────────────────────────────

type analyzeResult struct {
	OfficialName string `json:"official_name"`
	Description  string `json:"description"`
}

// ─── Query: Wikipedia lookup → LLM analysis → version discovery ───

// ProgressFunc is an optional callback for reporting query progress.
// step is a stable identifier (e.g. "wikipedia_search"), status is "running"/"done"/"error".
type ProgressFunc func(step, status, message string)

func (s *LanguageInitService) Query(ctx context.Context, languageName string) (*InitSuggestion, error) {
	return s.QueryWithProgress(ctx, languageName, nil)
}

func (s *LanguageInitService) QueryWithProgress(ctx context.Context, languageName string, progress ProgressFunc) (*InitSuggestion, error) {
	emit := func(step, status, message string) {
		if progress != nil {
			progress(step, status, message)
		}
	}

	// ── 步骤 1: Wikipedia 搜索 ──
	emit("wikipedia_search", "running", "正在搜索 Wikipedia…")
	hits, err := s.Scraper.SearchWikipedia(ctx, languageName+" programming language")
	if err != nil {
		emit("wikipedia_search", "error", err.Error())
		return nil, fmt.Errorf("Wikipedia 搜索失败: %w", err)
	}
	if len(hits) == 0 {
		emit("wikipedia_search", "error", "未找到匹配条目")
		return nil, fmt.Errorf("未找到 Wikipedia 条目: %q", languageName)
	}
	page := hits[0]
	emit("wikipedia_search", "done", "找到条目: "+page.Title)

	// ── 步骤 2: Wikipedia 分类验证 ──
	emit("wikipedia_categories", "running", "正在获取分类…")
	cats, errCat := s.Scraper.GetPageCategories(ctx, page.Title)
	if errCat != nil {
		emit("wikipedia_categories", "error", errCat.Error())
		return nil, fmt.Errorf("Wikipedia 分类获取失败: %w", errCat)
	}
	emit("wikipedia_categories", "done", fmt.Sprintf("找到 %d 个分类", len(cats)))

	// ── 步骤 3: Wikipedia 信息框解析 ──
	emit("wikipedia_infobox", "running", "正在解析信息框…")
	info, errInfo := s.Scraper.GetInfobox(ctx, page.Title)
	if errInfo != nil {
		emit("wikipedia_infobox", "error", errInfo.Error())
		return nil, fmt.Errorf("Wikipedia 信息框解析失败: %w", errInfo)
	}

	_, reject := scraper.ScoreSignal(cats, info)
	if reject {
		emit("wikipedia_infobox", "error", "不是编程语言")
		return nil, fmt.Errorf("不是编程语言: %q 被归类为非语言类型", page.Title)
	}
	emit("wikipedia_infobox", "done", "信息框解析完成")

	// ── 步骤 4: LLM 分析 ──
	emit("llm_analyze", "running", "LLM 正在分析语言分类与版本信息…")
	normalizedTitle := scraper.NormalizeTitle(page.Title)
	analysis, err := s.llmAnalyze(ctx, normalizedTitle, cats, info)
	if err != nil {
		emit("llm_analyze", "error", err.Error())
		return nil, fmt.Errorf("LLM 分析失败: %w", err)
	}
	emit("llm_analyze", "done", "分析完成: "+analysis.OfficialName)

	compatModel := scraper.CompatibilityModel(analysis.OfficialName, cats, info)
	slug := scraper.NormalizeSlug(analysis.OfficialName)
	icon := fetchIcon(ctx, s.Scraper, page.Title, analysis.OfficialName)

	var versions []DiscoveredVersion
	var latestVer string

	// Wikipedia infobox fast path
	if info != nil && info.LatestVersion != "" {
		v := cleanWikiVersion(info.LatestVersion)
		if v != "" {
			versions = append(versions, DiscoveredVersion{
				Version:  v,
				Released: info.FirstAppeared,
				Brief:    fmt.Sprintf("Latest stable version of %s", analysis.OfficialName),
				Source:   "wikipedia",
			})
		}
	}

	// ── 步骤 5: Wikipedia 外部链接收集 ──
	emit("wikipedia_links", "running", "正在收集 Wikipedia 外部链接…")
	var wikiLinks []string
	if s.Scraper != nil {
		links, err := s.Scraper.GetExternalLinks(ctx, page.Title)
		if err != nil {
			slog.Warn("wikipedia external links failed", "error", err)
		}
		wikiLinks = links
	}
	emit("wikipedia_links", "done", fmt.Sprintf("收集到 %d 个外部链接", len(wikiLinks)))

	// ── 步骤 6: 版本发现（结构化源 → HTML LLM 提取） ──
	emit("version_discovery", "running", "正在从 Docker Hub / GitHub / 官网 发现版本…")
	websiteURL := ""
	if info != nil {
		websiteURL = info.Website
	}
	discovered, err := s.discoverVersionsFromSource(ctx, analysis.OfficialName, slug, compatModel, websiteURL, wikiLinks)
	if err != nil {
		slog.Warn("version discovery from web failed",
			"language", analysis.OfficialName, "error", err)
	}
	if len(discovered) > 0 {
		versions = append(versions, discovered...)
		versions = dedupVersions(versions)
		versions = filterValidVersions(versions)
		if compatModel == "strict" && len(versions) > 1 {
			versions = versions[:1]
		}
	}
	emit("version_discovery", "done", fmt.Sprintf("发现 %d 个版本", len(versions)))

	if len(versions) == 0 && compatModel == "strict" {
		return nil, fmt.Errorf("strict language %q: no versions could be discovered", analysis.OfficialName)
	}

	if len(versions) > 0 {
		latestVer = versions[0].Version
	}

	return &InitSuggestion{
		Name:               analysis.OfficialName,
		Slug:               slug,
		Icon:               icon,
		CompatibilityModel: compatModel,
		Description:        analysis.Description,
		Versions:           versions,
		LatestVersion:      latestVer,
	}, nil
}

func (s *LanguageInitService) llmAnalyze(ctx context.Context, title string, cats []string, info *scraper.InfoboxData) (*analyzeResult, error) {
	// Step 1: Extract official name only (narrow task, minimal reasoning).
	tmpl1, err := llm.LoadTemplate(s.PromptDir+"/language_analyze.yaml", map[string]string{"Title": title})
	if err != nil {
		return nil, fmt.Errorf("load name template: %w", err)
	}
	content, _, err := s.LLM.ChatWithTemp(ctx, tmpl1.SystemPrompt, tmpl1.UserPrompt, tmpl1.Temperature, tmpl1.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("extract name: %w", err)
	}
	var result analyzeResult
	if err := llm.ParseLLMJSON(content, &result); err != nil {
		return nil, fmt.Errorf("parse name: %w", err)
	}
	if result.OfficialName == "" {
		return nil, fmt.Errorf("llm returned empty official name")
	}

	// Step 2: Generate description from infobox data (separate narrow call).
	descVars := map[string]string{"OfficialName": result.OfficialName}
	if info != nil {
		if info.InfoboxType != "" {
			descVars["InfoboxType"] = info.InfoboxType
		}
		if info.Developer != "" {
			descVars["Developer"] = info.Developer
		}
		if info.FirstAppeared != "" {
			descVars["FirstAppeared"] = info.FirstAppeared
		}
	}
	tmpl2, err := llm.LoadTemplate(s.PromptDir+"/language_describe.yaml", descVars)
	if err != nil {
		// Description is optional; continue without it.
		slog.Warn("failed to load description template, skipping description", "error", err)
		return &result, nil
	}
	descContent, _, descErr := s.LLM.ChatWithTemp(ctx, tmpl2.SystemPrompt, tmpl2.UserPrompt, tmpl2.Temperature, tmpl2.MaxTokens)
	if descErr != nil {
		slog.Warn("failed to generate description, continuing without it", "error", descErr)
		return &result, nil
	}
	var descResult struct {
		Description string `json:"description"`
	}
	if err := llm.ParseLLMJSON(descContent, &descResult); err != nil {
		slog.Warn("failed to parse description, continuing without it", "error", err)
		return &result, nil
	}
	result.Description = descResult.Description

	return &result, nil
}

// ─── Confirm ──────────────────────────────────────────────

func fetchIcon(ctx context.Context, client *scraper.Client, pageTitle, officialName string) string {
	if client == nil {
		return ""
	}
	img, err := client.GetPageImage(ctx, pageTitle)
	if err == nil && img != "" {
		return img
	}
	return ""
}

var validCompatModels = map[string]bool{
	"strict": true, "versioned": true, "none": true,
}

var slugRx = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func (s *LanguageInitService) Confirm(ctx context.Context, input InitConfirmInput) (*InitResult, error) {
	if input.Name == "" || input.Slug == "" || input.CompatibilityModel == "" {
		return nil, fmt.Errorf("name, slug, and compatibility_model are required")
	}
	if !validCompatModels[input.CompatibilityModel] {
		return nil, fmt.Errorf("invalid compatibility_model %q", input.CompatibilityModel)
	}
	if !slugRx.MatchString(input.Slug) {
		return nil, fmt.Errorf("invalid slug %q", input.Slug)
	}
	if input.DocsURL != "" {
		if u := validateURL(input.DocsURL); u == "" {
			return nil, fmt.Errorf("invalid docs_url: %s", input.DocsURL)
		}
	}
	if input.RuntimeURL != "" {
		if u := validateURL(input.RuntimeURL); u == "" {
			return nil, fmt.Errorf("invalid runtime_url: %s", input.RuntimeURL)
		}
	}

	sourceURLs, err := json.Marshal(map[string]string{"docs": input.DocsURL, "runtime": input.RuntimeURL})
	if err != nil {
		return nil, fmt.Errorf("marshal source_urls: %w", err)
	}

	lang, err := s.LangSvc.Create(ctx, CreateLanguageInput{
		Name:               input.Name,
		Slug:               input.Slug,
		Icon:               input.Icon,
		CompatibilityModel: input.CompatibilityModel,
		SourceURLs:         sourceURLs,
	})
	if err != nil {
		return nil, fmt.Errorf("create language: %w", err)
	}

	var createdVersions []model.LanguageVersion
	var initializedVersions []string
	if s.VersionSvc != nil && len(input.Versions) > 0 {
		discovered := make(map[string]*DiscoveredVersion)
		for i := range input.DiscoveredVersions {
			discovered[input.DiscoveredVersions[i].Version] = &input.DiscoveredVersions[i]
		}
		for _, ver := range input.Versions {
			v := &model.LanguageVersion{
				LanguageID: lang.ID,
				Version:    ver,
				Status:     "active",
			}
			var dv *DiscoveredVersion
			if d, ok := discovered[ver]; ok {
				dv = d
				srcData, _ := json.Marshal(map[string]interface{}{
					"download_url": dv.DownloadURL,
					"source_page":  dv.Source,
					"image_tag":    dv.ImageTag,
					"docker_refs":  dv.DockerRefs,
				})
				v.SourceURLs = srcData
			}
			if err := s.VersionSvc.Create(ctx, v); err != nil {
				slog.Warn("failed to create version", "language", lang.Name, "version", ver, "error", err)
				continue
			}
			if s.InitVersionSvc != nil && dv != nil {
				if s.autoInitialize(ctx, v, lang, dv) {
					initializedVersions = append(initializedVersions, ver)
				}
			}
			createdVersions = append(createdVersions, *v)
		}
	}

	return &InitResult{Language: lang, Versions: createdVersions, InitializedVersions: initializedVersions}, nil
}

// ─── Helpers ─────────────────────────────────────────────────

func cleanWikiVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "X.Y" || raw == "x.y" {
		return ""
	}
	re := regexp.MustCompile(`\d+(?:\.\d+)*`)
	matches := re.FindAllString(raw, -1)
	if len(matches) == 0 {
		return ""
	}
	for _, m := range matches {
		if len(m) >= 2 {
			return m
		}
	}
	return matches[0]
}

func resolveImage(dv *DiscoveredVersion, slug string) string {
	if dv != nil && dv.ImageTag != "" {
		return dv.ImageTag
	}
	if dv != nil {
		majorMinor := extractMajorMinor(dv.Version)
		for _, ref := range dv.DockerRefs {
			if strings.Contains(ref, majorMinor) {
				return ref
			}
		}
	}
	if base, ok := executor.DefaultImage(slug); ok && base != "" {
		if dv != nil {
			majorMinor := extractMajorMinor(dv.Version)
			if majorMinor != "" {
				re := regexp.MustCompile(`:\d+\.\d+`)
				if re.MatchString(base) {
					return re.ReplaceAllString(base, ":"+majorMinor)
				}
			}
		}
		return base
	}
	if dv != nil {
		return fmt.Sprintf("%s:%s", slug, dv.Version)
	}
	return fmt.Sprintf("%s:latest", slug)
}

func (s *LanguageInitService) autoInitialize(ctx context.Context, v *model.LanguageVersion, lang *model.Language, dv *DiscoveredVersion) bool {
	initCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	result, err := s.InitVersionSvc.Initialize(initCtx, v.ID)
	if err != nil {
		slog.Warn("auto-initialize failed", "language", lang.Name, "version", v.Version, "error", err)
		return false
	}
	slog.Info("auto-initialize ready", "language", lang.Name, "version", v.Version, "status", result.Status)

	// After runtime initialization succeeds, trigger KB build synchronously.
	// This is part of the creation flow — the user sees kb_status progress
	// on the version detail page.
	if s.KBBuildSvc != nil {
		kbCtx, kbCancel := context.WithTimeout(ctx, 10*time.Minute)
		defer kbCancel()
		slog.Info("starting KB build", "language", lang.Name, "version", v.Version)
		if kbErr := s.KBBuildSvc.Build(kbCtx, v.ID); kbErr != nil {
			slog.Warn("KB build failed", "language", lang.Name, "version", v.Version, "error", kbErr)
		} else {
			slog.Info("KB build complete", "language", lang.Name, "version", v.Version)
		}
	}

	return true
}

func validateURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	if strings.Contains(raw, "{") || strings.Contains(raw, "}") ||
		strings.Contains(raw, ".mw-") || strings.Contains(raw, "plainlist") {
		return ""
	}
	return u.String()
}

// splitIntoChunks splits text into overlapping chunks of at most chunkSize bytes.
func splitIntoChunks(text string, chunkSize, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}
	var chunks []string
	start := 0
	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[start:end])
		start += chunkSize - overlap
		if start >= len(text) {
			break
		}
	}
	return chunks
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ─── Version Discovery (Tools-First) ───────────────────────────

func (s *LanguageInitService) discoverVersionsFromSource(ctx context.Context, name, slug, compatModel, websiteURL string, wikiLinks []string) ([]DiscoveredVersion, error) {
	var allVersions []DiscoveredVersion

	candidates := buildCandidateSources(websiteURL, wikiLinks, slug)
	seen := make(map[string]bool)

	// Phase A: structured sources — zero LLM cost, just HTTP + JSON
	for _, c := range candidates {
		if c.sourceType != "structured" || seen[c.url] {
			continue
		}
		seen[c.url] = true
		versions := s.fetchStructuredVersions(ctx, c.url)
		for i := range versions {
			versions[i].Source = c.url
		}
		allVersions = append(allVersions, versions...)
	}
	if len(allVersions) > 0 && compatModel == "strict" {
		return s.postProcessVersions(allVersions, compatModel, slug), nil
	}

	// Phase B: HTML pages with LLM extraction.
	// Process ALL candidate pages — no artificial limit on LLM calls.
	// Large pages are automatically chunked into multiple smaller calls.
	for _, c := range candidates {
		if c.sourceType == "structured" || seen[c.url] {
			continue
		}
		seen[c.url] = true

		versions := s.extractVersionsFromHTML(ctx, name, slug, compatModel, c.url)
		for i := range versions {
			versions[i].Source = c.url
		}
		allVersions = append(allVersions, versions...)

		if compatModel == "strict" && len(allVersions) >= 1 {
			break
		}
	}

	if len(allVersions) == 0 {
		return nil, fmt.Errorf("no version info extracted from homepage or %d candidate sources", len(candidates))
	}

	return s.postProcessVersions(allVersions, compatModel, slug), nil
}

type sourceCandidate struct {
	url        string
	sourceType string
}

func buildCandidateSources(websiteURL string, wikiLinks []string, slug string) []sourceCandidate {
	var candidates []sourceCandidate

	// Docker Hub API — external data source, zero LLM cost.
	// Works for any language with an official Docker image; returns 404/empty for others.
	dockerHubURL := fmt.Sprintf("https://hub.docker.com/v2/repositories/library/%s/tags?page_size=25", slug)
	candidates = append(candidates, sourceCandidate{url: dockerHubURL, sourceType: "structured"})

	for _, link := range wikiLinks {
		if isWikipediaURL(link) || link == "" {
			continue
		}
		candidates = append(candidates, sourceCandidate{url: link, sourceType: classifySourceType(link)})
	}
	if websiteURL != "" {
		base := strings.TrimRight(websiteURL, "/")
		for _, p := range []string{"/releases", "/downloads"} {
			candidates = append(candidates, sourceCandidate{url: base + p, sourceType: "html"})
		}
	}
	for _, c := range candidates {
		if strings.Contains(c.url, "github.com/") && !strings.Contains(c.url, "api.github.com") {
			if apiURL := githubRepoToAPI(c.url); apiURL != "" {
				candidates = append([]sourceCandidate{{url: apiURL, sourceType: "structured"}}, candidates...)
			}
		}
	}
	sortCandidates(candidates)
	return candidates
}

func classifySourceType(urlStr string) string {
	lower := strings.ToLower(urlStr)
	if strings.Contains(lower, "/api/") ||
		strings.HasSuffix(lower, ".json") ||
		strings.HasSuffix(lower, ".rss") {
		return "structured"
	}
	return "html"
}

func githubRepoToAPI(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 {
		return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", parts[0], parts[1])
	}
	return ""
}

func isWikipediaURL(link string) bool {
	return strings.Contains(strings.ToLower(link), "wikipedia.org") ||
		strings.Contains(strings.ToLower(link), "wikimedia.org")
}

func sortCandidates(candidates []sourceCandidate) {
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].sourceType != "structured" && candidates[j].sourceType == "structured" {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
}

func (s *LanguageInitService) postProcessVersions(allVersions []DiscoveredVersion, compatModel string, slug string) []DiscoveredVersion {
	allVersions = dedupVersions(allVersions)
	allVersions = filterValidVersions(allVersions)

	if compatModel == "strict" && len(allVersions) > 1 {
		allVersions = allVersions[:1]
	}

	for i := range allVersions {
		if allVersions[i].DownloadURL != "" && !urlIsReachable(context.Background(), allVersions[i].DownloadURL) {
			allVersions[i].DownloadURL = ""
		}
	}

	for i := range allVersions {
		if allVersions[i].ImageTag != "" {
			continue
		}
		if len(allVersions[i].DockerRefs) > 0 {
			majorMinor := extractMajorMinor(allVersions[i].Version)
			for _, ref := range allVersions[i].DockerRefs {
				if strings.Contains(ref, majorMinor) {
					allVersions[i].ImageTag = ref
					break
				}
			}
		}
		if allVersions[i].ImageTag == "" {
			allVersions[i].ImageTag = resolveImage(&allVersions[i], slug)
		}
	}

	return allVersions
}

func (s *LanguageInitService) fetchStructuredVersions(ctx context.Context, u string) []DiscoveredVersion {
	raw, err := s.Scraper.FetchRaw(ctx, u)
	if err != nil {
		slog.Debug("structured: failed to fetch", "url", u, "error", err)
		return nil
	}
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}
	return extractVersionsFromJSON(data)
}

func extractVersionsFromJSON(data interface{}) []DiscoveredVersion {
	var versions []DiscoveredVersion
	extractVersionsRecursive(data, &versions)
	return dedupVersions(versions)
}

// extractVersionsRecursive walks arbitrary JSON structures looking for version strings.
// It does NOT assume any specific API format — it looks for objects containing
// version-like values (matching ^\d+\.\d+) in fields named version, name, tag_name,
// or any string value that looks like a version.
func extractVersionsRecursive(node interface{}, versions *[]DiscoveredVersion) {
	switch n := node.(type) {
	case map[string]interface{}:
		// Check if this object has a version-like field.
		for _, key := range []string{"version", "name", "tag_name", "tag", "ref", "release"} {
			if val, ok := n[key].(string); ok && looksLikeVersion(strings.TrimPrefix(val, "v")) {
				ver := strings.TrimPrefix(strings.TrimPrefix(val, "v"), "go")
				ver = cleanTag(ver)
				if looksLikeVersion(ver) {
					// Skip prerelease/snapshot markers
					skip := false
					for _, mk := range []string{"prerelease", "draft", "unstable"} {
						if b, ok := n[mk].(bool); ok && b {
							skip = true
							break
						}
					}
					if !skip {
						dv := DiscoveredVersion{Version: ver}
						if _, ok := n["name"].(string); ok {
							dv.ImageTag = val
						}
						*versions = append(*versions, dv)
					}
				}
			}
		}
		// Check keys that might be version numbers themselves (e.g. "3.12.0": {...})
		for k, v := range n {
			if looksLikeVersion(k) {
				if obj, ok := v.(map[string]interface{}); ok {
					extractVersionsRecursive(obj, versions)
				} else {
					*versions = append(*versions, DiscoveredVersion{Version: k})
				}
			}
		}
		// Recurse into all values
		for _, v := range n {
			extractVersionsRecursive(v, versions)
		}
	case []interface{}:
		for _, item := range n {
			extractVersionsRecursive(item, versions)
		}
	}
}

func looksLikeVersion(s string) bool { return regexp.MustCompile(`^\d+\.\d+`).MatchString(s) }

func cleanTag(tag string) string {
	if idx := strings.IndexAny(tag, "-_"); idx > 0 {
		return tag[:idx]
	}
	return tag
}

func sortVersionsByLatest(versions []DiscoveredVersion, latest string) {
	for i := range versions {
		if versions[i].Version == latest && i > 0 {
			versions[0], versions[i] = versions[i], versions[0]
			break
		}
	}
}

func (s *LanguageInitService) extractVersionsFromHTML(ctx context.Context, name, slug, compatModel, u string) []DiscoveredVersion {
	return s.extractVersionsFromHTMLPage(ctx, name, slug, compatModel, u)
}

// extractVersionsFromHTMLPage fetches a page and extracts versions.
// For large pages, it splits the text into overlapping chunks and makes
// multiple LLM calls — one per chunk — then merges the results.
func (s *LanguageInitService) extractVersionsFromHTMLPage(ctx context.Context, name, slug, compatModel, u string) []DiscoveredVersion {
	text, err := s.Scraper.FetchPageText(ctx, u, 30000)
	if err != nil {
		slog.Debug("extract: failed to fetch page", "url", u, "error", err)
		return nil
	}

	const maxChunk = 6000
	const overlap = 400

	if len(text) <= maxChunk {
		vs, dockerRefs := s.extractVersionsFromChunk(ctx, name, slug, compatModel, u, text, false)
		for i := range vs {
			vs[i].DockerRefs = dockerRefs
		}
		return vs
	}

	chunks := splitIntoChunks(text, maxChunk, overlap)
	slog.Debug("extract: chunking large page",
		"url", u, "text_len", len(text), "chunks", len(chunks))

	var all []DiscoveredVersion
	var allDockerRefs []string
	for _, chunk := range chunks {
		vs, dockerRefs := s.extractVersionsFromChunk(ctx, name, slug, compatModel, u, chunk, true)
		for j := range vs {
			vs[j].Source = u
		}
		all = append(all, vs...)
		allDockerRefs = append(allDockerRefs, dockerRefs...)
	}

	// If we got versions from multiple chunks, do one more merge pass
	// to deduplicate and reconcile.
	if len(all) > 0 && len(chunks) > 2 {
		all = s.mergeChunkedVersions(ctx, name, slug, compatModel, u, all)
	}

	// Apply docker refs to all entries that don't have one
	uniqueRefs := uniqueStrings(allDockerRefs)
	if len(uniqueRefs) > 0 {
		for i := range all {
			if len(all[i].DockerRefs) == 0 {
				all[i].DockerRefs = uniqueRefs
			}
		}
	}

	return dedupVersions(all)
}

// extractVersionsFromChunk calls LLM to extract versions from a text chunk.
func (s *LanguageInitService) extractVersionsFromChunk(ctx context.Context, name, slug, compatModel, pageURL, text string, isChunk bool) ([]DiscoveredVersion, []string) {
	isChunkStr := "false"
	if isChunk {
		isChunkStr = "true"
	}

	tmpl, err := llm.LoadTemplate(s.PromptDir+"/extract_versions.yaml", map[string]string{
		"LanguageName":       name,
		"Slug":               slug,
		"CompatibilityModel": compatModel,
		"PageURL":            pageURL,
		"PageText":           text,
		"IsChunk":            isChunkStr,
	})
	if err != nil {
		slog.Debug("extract: failed to load template", "error", err)
		return nil, nil
	}

	content, _, err := s.LLM.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		slog.Debug("extract: llm call failed", "error", err)
		return nil, nil
	}

	var result struct {
		Versions   []DiscoveredVersion `json:"versions"`
		Latest     string              `json:"latest"`
		DockerRefs []string            `json:"docker_refs"`
	}
	if err := llm.ParseLLMJSON(content, &result); err != nil {
		slog.Debug("extract: parse failed", "error", err)
		return nil, nil
	}
	return result.Versions, result.DockerRefs
}

// mergeChunkedVersions makes one final LLM call to reconcile versions
// extracted from multiple chunks, removing duplicates and normalizing.
func (s *LanguageInitService) mergeChunkedVersions(ctx context.Context, name, slug, compatModel, pageURL string, versions []DiscoveredVersion) []DiscoveredVersion {
	// Build a compact list for the merge prompt
	var compact []map[string]string
	for _, v := range versions {
		compact = append(compact, map[string]string{
			"version":      v.Version,
			"lts":          fmt.Sprintf("%v", v.LTS),
			"released":     v.Released,
			"download_url": v.DownloadURL,
		})
	}
	compactJSON, _ := json.Marshal(compact)

	tmpl, err := llm.LoadTemplate(s.PromptDir+"/extract_versions_merge.yaml", map[string]string{
		"LanguageName":       name,
		"Slug":               slug,
		"CompatibilityModel": compatModel,
		"PageURL":            pageURL,
		"ChunkedVersions":    string(compactJSON),
	})
	if err != nil {
		slog.Debug("merge: failed to load template", "error", err)
		return versions
	}

	content, _, err := s.LLM.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, 0.1, 2048)
	if err != nil {
		slog.Debug("merge: llm call failed", "error", err)
		return versions
	}

	var merged []DiscoveredVersion
	if err := llm.ParseLLMJSON(content, &merged); err != nil {
		slog.Debug("merge: parse failed", "error", err)
		return versions
	}
	return merged
}

func dedupVersions(versions []DiscoveredVersion) []DiscoveredVersion {
	seen := make(map[string]bool)
	var out []DiscoveredVersion
	for _, v := range versions {
		if !seen[v.Version] {
			seen[v.Version] = true
			out = append(out, v)
		}
	}
	return out
}

func filterValidVersions(versions []DiscoveredVersion) []DiscoveredVersion {
	hasDigit := regexp.MustCompile(`\d`)
	var out []DiscoveredVersion
	for _, v := range versions {
		if v.Version != "" && v.Version != "X.Y" && v.Version != "x.y" && hasDigit.MatchString(v.Version) {
			out = append(out, v)
		}
	}
	return out
}

func urlIsReachable(ctx context.Context, rawURL string) bool {
	if rawURL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "LearnCode/1.0")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func extractMajorMinor(version string) string {
	m := regexp.MustCompile(`^(\d+\.\d+)`).FindString(version)
	return m
}

// ─── Research ──────────────────────────────────────────────────

type ResourceEntry struct {
	URL         string `json:"url"`
	Authority   string `json:"authority"`
	Description string `json:"description"`
}

type ResearchResult struct {
	Docs     []ResourceEntry `json:"docs"`
	Runtimes []ResourceEntry `json:"runtimes"`
	Specs    []ResourceEntry `json:"specs"`
}

func (s *LanguageInitService) Research(ctx context.Context, lang *model.Language) (*ResearchResult, error) {
	if s.LLM == nil {
		return nil, fmt.Errorf("llm service not available")
	}

	var website string
	var externalLinks []string

	if s.Scraper != nil {
		info, _ := s.Scraper.GetInfobox(ctx, lang.Name)
		if info != nil && info.Website != "" {
			website = info.Website
		}
		links, _ := s.Scraper.GetExternalLinks(ctx, lang.Name)
		if len(links) > 0 {
			externalLinks = links
		}
	}

	pageText := ""
	if website != "" && s.Scraper != nil {
		text, err := s.Scraper.FetchPageText(ctx, website, 15000)
		if err == nil {
			pageText = text
		}
	}

	vars := map[string]string{
		"OfficialName":  lang.Name,
		"WebsiteURL":    website,
		"PageText":      pageText,
		"ExternalLinks": strings.Join(externalLinks, "\n"),
	}

	tmpl, err := llm.LoadTemplate(s.PromptDir+"/language_research.yaml", vars)
	if err != nil {
		return nil, fmt.Errorf("load research template: %w", err)
	}

	content, _, err := s.LLM.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("llm research: %w", err)
	}

	var llmResult struct {
		Docs     []ResourceEntry `json:"docs"`
		Runtimes []ResourceEntry `json:"runtimes"`
		Specs    []ResourceEntry `json:"specs"`
	}
	if err := llm.ParseLLMJSON(content, &llmResult); err != nil {
		return nil, fmt.Errorf("parse research response: %w", err)
	}

	return &ResearchResult{
		Docs:     llmResult.Docs,
		Runtimes: llmResult.Runtimes,
		Specs:    llmResult.Specs,
	}, nil
}
