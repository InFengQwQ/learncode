package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"learncode/internal/llm"
	"learncode/internal/model"
	"learncode/internal/scraper"
)

type InitSuggestion struct {
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Icon               string `json:"icon"`
	CompatibilityModel string `json:"compatibility_model"`
	Description        string `json:"description"`
	DocsURL            string `json:"docs_url"`
	RuntimeURL         string `json:"runtime_url"`
	Confidence         int    `json:"confidence"`
	Reasoning          string `json:"reasoning,omitempty"`
}

type InitConfirmInput struct {
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Icon               string `json:"icon"`
	CompatibilityModel string `json:"compatibility_model"`
	DocsURL            string `json:"docs_url"`
	RuntimeURL         string `json:"runtime_url"`
}

type InitResult struct {
	Language *model.Language `json:"language"`
}

type LanguageInitService struct {
	LangSvc   *LanguageService
	LLM       *llm.Service
	PromptDir string
	Scraper   *scraper.Client
}

// ─── Internal LLM response types ──────────────────────────────

type analyzeResult struct {
	IsLanguage   bool   `json:"is_language"`
	OfficialName string `json:"official_name"`
	Description  string `json:"description"`
	Confidence   int    `json:"confidence"`
	Reasoning    string `json:"reasoning,omitempty"`
}

type resourceResult struct {
	DocsURL        string `json:"docs_url"`
	DocsAuthority  string `json:"docs_authority"`
	RuntimeURL     string `json:"runtime_url"`
	RuntimeExists  bool   `json:"runtime_exists"`
}

// ─── Query: Wikipedia lookup → LLM analysis → resource identification ───

func (s *LanguageInitService) Query(ctx context.Context, languageName string) (*InitSuggestion, error) {
	// Step 1: Wikipedia search
	hits, err := s.Scraper.SearchWikipedia(ctx, languageName+" programming language")
	if err != nil {
		return nil, fmt.Errorf("wikipedia search: %w", err)
	}
	if len(hits) == 0 {
		return nil, fmt.Errorf("language not found: no Wikipedia article for %q", languageName)
	}
	page := hits[0]

	// Normalize the Wikipedia title — strip disambiguation like
	// "Python (programming language)" → "Python" before feeding to the LLM.
	normalizedTitle := scraper.NormalizeTitle(page.Title)

	// Step 2: Get categories + infobox
	cats, _ := s.Scraper.GetPageCategories(ctx, page.Title)
	info, _ := s.Scraper.GetInfobox(ctx, page.Title)

	// Step 3: Signal scoring — reject obvious non-languages
	_, reject := scraper.ScoreSignal(cats, info)
	if reject {
		return nil, fmt.Errorf("not a programming language: %q is classified as something else", page.Title)
	}

	// Step 4: LLM analysis — the LLM only decides is_language, official_name,
	// description, and confidence. Everything else is deterministic Go code below.
	analysis, err := s.llmAnalyze(ctx, normalizedTitle, cats, info)
	if err != nil {
		return nil, fmt.Errorf("llm analysis: %w", err)
	}
	if !analysis.IsLanguage || analysis.Confidence < 5 {
		return nil, fmt.Errorf("not a programming language: %s", analysis.Reasoning)
	}

	// Step 4b: Go-level determinations — these are NOT from the LLM.
	// Small models produce inconsistent results for these fields.
	compatModel := scraper.CompatibilityModel(analysis.OfficialName, cats, info)
	slug := scraper.NormalizeSlug(analysis.OfficialName)

	// Icon: try Wikipedia page image first, fall back to emoji lookup.
	// Use the original page title (with disambiguation) for the image API
	// because that's the actual Wikipedia article name.
	icon := fetchIcon(ctx, s.Scraper, page.Title, analysis.OfficialName)

	// Step 5: Resource identification — fetch official site and have LLM extract URLs
	var docsURL, runtimeURL string
	if info != nil && info.Website != "" {
		pageText, err := s.Scraper.FetchPageText(ctx, info.Website)
		if err == nil && pageText != "" {
			res, _ := s.llmResources(ctx, analysis.OfficialName, info.Website, pageText)
			if res != nil && res.RuntimeExists {
				docsURL = res.DocsURL
				runtimeURL = res.RuntimeURL
			}
		}
	}
	// Fallback: use Wikipedia data directly
	if docsURL == "" && info != nil {
		docsURL = info.Website
	}
	if runtimeURL == "" && info != nil {
		runtimeURL = info.Website
	}

	// Validate URLs before sending to frontend — reject CSS garbage, file://, etc.
	docsURL = validateURL(docsURL)
	runtimeURL = validateURL(runtimeURL)

	return &InitSuggestion{
		Name:               analysis.OfficialName,
		Slug:               slug,
		Icon:               icon,
		CompatibilityModel: compatModel,
		Description:        analysis.Description,
		DocsURL:            docsURL,
		RuntimeURL:         runtimeURL,
		Confidence:         analysis.Confidence,
		Reasoning:          analysis.Reasoning,
	}, nil
}

func (s *LanguageInitService) llmAnalyze(ctx context.Context, title string, cats []string, info *scraper.InfoboxData) (*analyzeResult, error) {
	vars := map[string]string{
		"Title": title,
	}
	if info != nil {
		if info.InfoboxType != "" {
			vars["InfoboxType"] = info.InfoboxType
		}
		if info.Website != "" {
			vars["Website"] = info.Website
		}
		if info.Developer != "" {
			vars["Developer"] = info.Developer
		}
		if info.FirstAppeared != "" {
			vars["FirstAppeared"] = info.FirstAppeared
		}
		if info.LatestVersion != "" {
			vars["LatestVersion"] = info.LatestVersion
		}
		if info.Typing != "" {
			vars["Typing"] = info.Typing
		}
	}
	if len(cats) > 0 {
		vars["Categories"] = strings.Join(cats, ", ")
	}

	tmpl, err := llm.LoadTemplate(s.PromptDir+"/language_analyze.yaml", vars)
	if err != nil {
		return nil, fmt.Errorf("load template: %w", err)
	}

	content, _, err := s.LLM.Chat(ctx, tmpl.SystemPrompt, tmpl.UserPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm chat: %w", err)
	}

	var result analyzeResult
	if err := parseLLMJSON(content, &result); err != nil {
		return nil, fmt.Errorf("parse llm response: %w", err)
	}
	return &result, nil
}

func (s *LanguageInitService) llmResources(ctx context.Context, name, website, pageText string) (*resourceResult, error) {
	tmpl, err := llm.LoadTemplate(s.PromptDir+"/language_resources.yaml", map[string]string{
		"OfficialName": name,
		"WebsiteURL":   website,
		"PageText":     pageText,
	})
	if err != nil {
		return nil, fmt.Errorf("load template: %w", err)
	}

	content, _, err := s.LLM.Chat(ctx, tmpl.SystemPrompt, tmpl.UserPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm chat: %w", err)
	}

	var result resourceResult
	if err := parseLLMJSON(content, &result); err != nil {
		return nil, fmt.Errorf("parse llm response: %w", err)
	}
	return &result, nil
}

// ─── Confirm: validate and create Language ──────────────────────────

// fetchIcon resolves a language icon: Wikipedia page image first, then emoji
// lookup by name, finally empty string (caller uses placeholder in UI).
func fetchIcon(ctx context.Context, client *scraper.Client, pageTitle, officialName string) string {
	if client == nil {
		return scraper.IconEmoji(officialName)
	}
	img, err := client.GetPageImage(ctx, pageTitle)
	if err != nil {
		// API failure — fall through to emoji
	} else if img != "" {
		return img
	}
	return scraper.IconEmoji(officialName)
}

var validCompatModels = map[string]bool{
	"strict":    true,
	"versioned": true,
	"none":      true,
}

var slugRx = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func (s *LanguageInitService) Confirm(ctx context.Context, input InitConfirmInput) (*InitResult, error) {
	// --- required fields ---
	if input.Name == "" || input.Slug == "" || input.CompatibilityModel == "" {
		return nil, fmt.Errorf("name, slug, and compatibility_model are required")
	}

	// --- validate compatibility_model enum ---
	if !validCompatModels[input.CompatibilityModel] {
		return nil, fmt.Errorf(
			"invalid compatibility_model %q: must be strict, versioned, or none",
			input.CompatibilityModel,
		)
	}

	// --- validate slug format (lowercase alphanumeric + hyphens, no leading/trailing hyphens, no consecutive hyphens) ---
	if !slugRx.MatchString(input.Slug) {
		return nil, fmt.Errorf(
			"invalid slug %q: must be lowercase letters, digits, and single hyphens only (e.g. java, cpp, c-sharp)",
			input.Slug,
		)
	}

	// --- validate URLs are well-formed ---
	if input.DocsURL != "" {
		if u := validateURL(input.DocsURL); u == "" {
			return nil, fmt.Errorf("invalid docs_url: %s", input.DocsURL)
		}
		input.DocsURL = validateURL(input.DocsURL)
	}
	if input.RuntimeURL != "" {
		if u := validateURL(input.RuntimeURL); u == "" {
			return nil, fmt.Errorf("invalid runtime_url: %s", input.RuntimeURL)
		}
		input.RuntimeURL = validateURL(input.RuntimeURL)
	}

	sourceURLs, err := json.Marshal(map[string]string{
		"docs":    input.DocsURL,
		"runtime": input.RuntimeURL,
	})
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

	return &InitResult{Language: lang}, nil
}

// ─── JSON parsing ─────────────────────────────────────────

// validateURL checks that a string is a well-formed http/https URL and
// returns the normalized URL string. Returns "" for invalid input.
// This catches CSS garbage, file:// URIs, and other non-URL content
// that may have leaked from infobox parsing.
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
	// Reject strings that look like CSS (contain braces, semicolons, or common CSS selectors)
	if strings.Contains(raw, "{") || strings.Contains(raw, "}") ||
		strings.Contains(raw, ".mw-") || strings.Contains(raw, "plainlist") {
		return ""
	}
	return u.String()
}

func parseLLMJSON(content string, v interface{}) error {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	idx := strings.Index(content, "{")
	if idx >= 0 {
		content = content[idx:]
	}
	return json.Unmarshal([]byte(content), v)
}
