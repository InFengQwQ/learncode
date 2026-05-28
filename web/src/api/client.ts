import { readNDJSONStream } from './stream'
import { requestDeduped } from '../lib/dedup'

const BASE = '/api/v1'
const REQUEST_TIMEOUT = 30000
const MAX_RETRIES = 2

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
  wiki_title?: string
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
  wiki_title?: string
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

function friendlyError(err: string): string {
  if (err.startsWith('Wikipedia') || err.startsWith('LLM') || err.startsWith('未找到')) return err
  if (err.includes('language not found')) return '未找到此名称的编程语言'
  if (err.includes('not a programming language')) return '这不是一门编程语言'
  if (err.includes('no Wikipedia article')) return '在 Wikipedia 中未找到此名称'
  if (err.includes('strict language') && err.includes('no versions')) return '未能发现任何版本，无法创建严格模式语言'
  if (err.includes('invalid compatibility_model')) return '兼容性模型无效'
  if (err.includes('invalid slug')) return '标识符格式无效'
  if (err.includes('network')) return '网络连接失败，请检查网络后重试'
  if (err.includes('timeout')) return '查询超时，请稍后重试'
  return err
}

function combineSignals(...signals: (AbortSignal | undefined)[]): AbortSignal {
  const controller = new AbortController()
  for (const signal of signals) {
    if (signal?.aborted) {
      controller.abort(signal.reason)
      return controller.signal
    }
    signal?.addEventListener('abort', () => controller.abort(signal.reason), { once: true })
  }
  return controller.signal
}

async function request<T>(path: string, options?: RequestInit): Promise<APIResponse<T>> {
  const timeoutController = new AbortController()
  const timeoutId = setTimeout(() => timeoutController.abort(new DOMException('Timeout', 'TimeoutError')), REQUEST_TIMEOUT)

  const combinedSignal = options?.signal
    ? combineSignals(options.signal, timeoutController.signal)
    : timeoutController.signal

  for (let attempt = 1; attempt <= MAX_RETRIES + 1; attempt++) {
    try {
      const res = await fetch(`${BASE}${path}`, {
        headers: { 'Content-Type': 'application/json' },
        ...options,
        signal: combinedSignal,
      })

      clearTimeout(timeoutId)

      if (res.status === 204) {
        return { ok: true, data: undefined }
      }

      const json: APIResponse<T> = await res.json()

      // Normalize error strings for client display
      if (!json.ok && json.error) {
        json.error = friendlyError(json.error)
      }

      // Retry on 5xx
      if (res.status >= 500 && attempt <= MAX_RETRIES) {
        await new Promise((r) => setTimeout(r, 1000 * attempt))
        continue
      }

      return json
    } catch (e) {
      // Don't retry aborted requests
      if (e instanceof DOMException && (e.name === 'AbortError' || e.name === 'TimeoutError')) {
        clearTimeout(timeoutId)
        return { ok: false, error: e.name === 'TimeoutError' ? '请求超时，请稍后重试' : '已取消' }
      }

      if (attempt <= MAX_RETRIES) {
        await new Promise((r) => setTimeout(r, 1000 * attempt))
        continue
      }
    }
  }

  clearTimeout(timeoutId)
  return { ok: false, error: '网络连接失败，请检查网络后重试' }
}

async function requestGet<T>(path: string): Promise<APIResponse<T>> {
  return requestDeduped(() => request<T>(path), path)
}

export interface ResourceEntry {
  url: string
  authority: string // "official" | "community"
  description: string
}

export interface ResearchResult {
  docs: ResourceEntry[] | null
  runtimes: ResourceEntry[] | null
  specs: ResourceEntry[] | null
}

export const languages = {
  list: () => requestGet<Language[]>('/languages'),
  get: (id: string) => requestGet<Language>(`/languages/${id}`),
  create: (input: CreateLanguageInput) =>
    request<Language>('/languages', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  delete: (id: string) =>
    request<void>(`/languages/${id}`, { method: 'DELETE' }),
  initQuery: (name: string, signal?: AbortSignal) =>
    request<InitSuggestion>('/languages/init?step=query', {
      method: 'POST',
      body: JSON.stringify({ name }),
      signal,
    }),

  // Streams query progress as NDJSON lines. Each line is a JSON object:
  //   {"step":"wikipedia_search","status":"running","message":"…"}
  //   {"step":"fatal","status":"error","message":"…"}  (on failure)
  //   {"ok":true,"data":{…}}  (final result)
  initQueryStream: async function* (name: string, signal?: AbortSignal): AsyncGenerator<Record<string, unknown>> {
    const res = await fetch(`${BASE}/languages/init?step=query&stream=true`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
      signal,
    })
    yield* readNDJSONStream(res, signal)
  },
  initConfirm: (input: InitConfirmInput, signal?: AbortSignal) =>
    request<InitResult>('/languages/init?step=confirm', {
      method: 'POST',
      body: JSON.stringify(input),
      signal,
    }),
  research: (id: string, signal?: AbortSignal) =>
    request<ResearchResult>(`/languages/${id}/research`, {
      method: 'POST',
      signal,
    }),
  discoverVersions: (id: string) =>
    request<LanguageVersion[]>(`/languages/${id}/discover-versions`, {
      method: 'POST',
    }),
  discoverVersionsStream: async function* (id: string, signal?: AbortSignal): AsyncGenerator<Record<string, unknown>> {
    const res = await fetch(`${BASE}/languages/${id}/discover-versions?stream=true`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      signal,
    })
    yield* readNDJSONStream(res, signal)
  },
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
    requestGet<LanguageVersion[]>(`/languages/${languageId}/versions`),
  get: (id: string) => requestGet<LanguageVersion>(`/versions/${id}`),
  create: (languageId: string, input: CreateVersionInput) =>
    request<LanguageVersion>(`/languages/${languageId}/versions`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  initialize: (versionId: string, signal?: AbortSignal) =>
    request<VersionInitResult>(`/versions/${versionId}/initialize`, {
      method: 'POST',
      signal,
    }),
  buildKnowledge: (versionId: string, signal?: AbortSignal) =>
    request<{ status: string; version_id: string }>(`/versions/${versionId}/build-knowledge`, {
      method: 'POST',
      signal,
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
      `/versions/${versionId}/knowledge`,
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
  run: (input: ExecuteInput, signal?: AbortSignal) =>
    request<ExecuteOutput>('/execute', {
      method: 'POST',
      body: JSON.stringify(input),
      signal,
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
  getLLM: () => requestGet<LLMConfig>('/config/llm'),
  updateLLM: (cfg: LLMConfig) =>
    request<LLMConfig>('/config/llm', {
      method: 'PUT',
      body: JSON.stringify(cfg),
    }),
}
