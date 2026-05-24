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
  created_at: string
  updated_at: string
}

export interface InitSuggestion {
  name: string
  slug: string
  icon: string
  compatibility_model: string
  description: string
  docs_url: string
  runtime_url: string
}

export interface InitConfirmInput {
  name: string
  slug: string
  icon: string
  compatibility_model: string
  docs_url: string
  runtime_url: string
}

export interface InitResult {
  language: Language
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
}

export const versions = {
  listByLanguage: (languageId: string) =>
    request<LanguageVersion[]>(`/languages/${languageId}/versions`),
  get: (id: string) => request<LanguageVersion>(`/versions/${id}`),
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
