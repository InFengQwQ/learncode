const BASE = '/api/v1'

export interface APIResponse<T> {
  ok: boolean
  data?: T
  error?: string
}

export interface Language {
  id: string
  name: string
  slug: string
  icon: string
  compatibility_model: string
  source_urls: Record<string, string>
  research_data: ResearchResult | null
  researched_at: string | null
  status: 'inactive' | 'active'
  created_at: string
}

export interface CreateLanguageInput {
  name: string
  slug: string
  compatibility_model: string
}

export interface LanguageVersion {
  id: string
  language_id: string
  version: string
  status: string
  runtime_config: unknown
  source_urls: unknown
  last_version_check_at: string | null
  initialized: boolean
  image: string
  kb_status: 'pending' | 'building' | 'complete' | 'failed'
  initialized_at: string | null
  created_at: string
  updated_at: string
}

export interface DiscoveredVersion {
  version: string
  lts: boolean
  released: string
  brief: string
  download_url?: string
  image_tag?: string
  source?: string
  docker_refs?: string[]
}

export interface InitSuggestion {
  name: string
  slug: string
  icon: string
  compatibility_model: string
  description: string
  /** @deprecated No longer required — environment sources are in versions */
  docs_url?: string
  /** @deprecated No longer required — environment sources are in versions */
  runtime_url?: string
  versions: DiscoveredVersion[]
  latest_version: string
  confidence: number
  reasoning?: string
}

export interface InitConfirmInput {
  name: string
  slug: string
  icon: string
  compatibility_model: string
  docs_url?: string
  runtime_url?: string
  versions: string[]
  discovered_versions?: DiscoveredVersion[]
}

export interface InitResult {
  language: Language
  versions: LanguageVersion[]
  initialized_versions: string[]
}

async function request<T>(path: string, options?: RequestInit): Promise<APIResponse<T>> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (res.status === 204) {
    return { ok: true, data: undefined }
  }
  return res.json()
}

export interface ResourceEntry {
  url: string
  authority: string // "official" | "community"
  description: string
}

export interface ResearchResult {
  docs: ResourceEntry[]
  runtimes: ResourceEntry[]
  specs: ResourceEntry[]
}

export const languages = {
  list: () => request<Language[]>('/languages'),
  get: (id: string) => request<Language>(`/languages/${id}`),
  create: (input: CreateLanguageInput) =>
    request<Language>('/languages', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  delete: (id: string) =>
    request<void>(`/languages/${id}`, { method: 'DELETE' }),
  initQuery: (name: string) =>
    request<InitSuggestion>('/languages/init?step=query', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  initConfirm: (input: InitConfirmInput) =>
    request<InitResult>('/languages/init?step=confirm', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  research: (id: string) =>
    request<ResearchResult>(`/languages/${id}/research`, {
      method: 'POST',
    }),
}

export interface CreateVersionInput {
  version: string
  status?: string
}

export interface VersionInitResult {
  status: 'success' | 'unavailable' | 'host_mode'
  message: string
  verified: boolean
  image_ref: string
}

export const versions = {
  listByLanguage: (languageId: string) =>
    request<LanguageVersion[]>(`/languages/${languageId}/versions`),
  get: (id: string) => request<LanguageVersion>(`/versions/${id}`),
  create: (languageId: string, input: CreateVersionInput) =>
    request<LanguageVersion>(`/languages/${languageId}/versions`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  initialize: (versionId: string) =>
    request<VersionInitResult>(`/versions/${versionId}/initialize`, {
      method: 'POST',
    }),
  buildKnowledge: (versionId: string) =>
    request<{ status: string; version_id: string }>(`/versions/${versionId}/build-knowledge`, {
      method: 'POST',
    }),
  setStatus: (versionId: string, status: 'active' | 'archived') =>
    request<LanguageVersion>(`/versions/${versionId}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    }),
}

export interface KnowledgeEntry {
  id: string
  language_id: string
  version_id: string | null
  scope: 'core' | 'version' | 'idiom'
  category: 'factual' | 'normative'
  topic: string
  content: Record<string, unknown>
  source: 'llm' | 'env'
  created_at: string
  updated_at: string
}

export const knowledge = {
  list: (versionId: string) =>
    request<{ shared: KnowledgeEntry[]; private: KnowledgeEntry[] }>(
      `/versions/${versionId}/knowledge`
    ),
}

export interface ExecuteInput {
  version_id: string
  code: string
}

export interface ExecuteOutput {
  stdout: string
  stderr: string
  exit_code: number
  duration_ms: number
}

export const execute = {
  run: (input: ExecuteInput) =>
    request<ExecuteOutput>('/execute', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
}

export interface LLMProvider {
  name: string
  endpoint: string
  models: string[]
  api_key: string
}

export interface LLMConfig {
  default: string
  providers: LLMProvider[]
}

export const config = {
  getLLM: () => request<LLMConfig>('/config/llm'),
  updateLLM: (cfg: LLMConfig) =>
    request<LLMConfig>('/config/llm', {
      method: 'PUT',
      body: JSON.stringify(cfg),
    }),
}
