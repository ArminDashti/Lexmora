const TOKEN_KEY = 'lexmora_token'
const USERNAME_KEY = 'lexmora_username'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setAuth(token: string, username: string) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USERNAME_KEY, username)
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USERNAME_KEY)
}

export function getUsername(): string | null {
  return localStorage.getItem(USERNAME_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

function withBase(path: string): string {
  const base = (import.meta.env.BASE_URL || '/').replace(/\/$/, '')
  if (!base || base === '/') return path
  return `${base}${path.startsWith('/') ? path : `/${path}`}`
}

function apiUrl(path: string): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') || ''
  const rel = withBase(`/api/v1${path}`)
  if (!apiBase) return rel
  return `${apiBase}${rel.startsWith('/') ? rel : `/${rel}`}`
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> | undefined),
  }

  const token = getToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const res = await fetch(apiUrl(path), { ...options, headers })

  if (res.status === 401) {
    clearAuth()
    window.location.href = `${(import.meta.env.BASE_URL || '/').replace(/\/$/, '')}/login`
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || 'Request failed')
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json()
}

export const api = {
  login: (username: string, password: string) =>
    request<{ token: string; username: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),

  transform: (payload: Record<string, unknown>) =>
    request<TransformResult>('/transform', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  getTransformOptions: () => request<TransformOptions>('/transform/options'),

  getHistory: (params: {
    sort_by?: string
    sort_order?: string
    type?: string | string[]
    from?: string
    to?: string
    limit?: number
    offset?: number
  }) => {
    const q = new URLSearchParams()
    if (params.sort_by) q.set('sort_by', params.sort_by)
    if (params.sort_order) q.set('sort_order', params.sort_order)
    if (params.type != null) {
      const types = Array.isArray(params.type)
        ? params.type.filter(Boolean)
        : params.type
          ? [params.type]
          : []
      if (types.length) q.set('type', types.join(','))
    }
    if (params.from) q.set('from', params.from)
    if (params.to) q.set('to', params.to)
    if (params.limit != null) q.set('limit', String(params.limit))
    if (params.offset != null) q.set('offset', String(params.offset))
    return request<HistoryPage>(`/history?${q}`)
  },

  createQuiz: (payload: { count: number }) =>
    request<QuizResponse>('/quiz', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  getHistoryItem: (id: string) => request<HistoryRecord>(`/history/${id}`),

  deleteHistory: (id: string) =>
    request<void>(`/history/${id}`, { method: 'DELETE' }),

  getStats: () => request<StatsResponse>('/stats'),

  getInstructions: () => request<Instruction[]>('/instructions'),

  getInstruction: (key: string) => request<Instruction>(`/instructions/${key}`),

  createInstruction: (payload: CreateInstructionPayload) =>
    request<Instruction>('/instructions', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  updateInstruction: (key: string, content: string) =>
    request<Instruction>(`/instructions/${key}`, {
      method: 'PUT',
      body: JSON.stringify({ content }),
    }),

  getSettings: () => request<AppSettings>('/settings'),

  updateSettings: (payload: Partial<AppSettings>) =>
    request<AppSettings>('/settings', {
      method: 'PATCH',
      body: JSON.stringify(payload),
    }),

  searchModels: (q: string) => {
    const params = new URLSearchParams()
    if (q) params.set('q', q)
    return request<OpenRouterModel[]>(`/settings/models?${params}`)
  },

  getCredits: () => request<CreditsInfo>('/settings/credits'),

  clearData: () => request<void>('/settings/data', { method: 'DELETE' }),
}

export interface TransformResult {
  id: string
  type: string
  type_display: string
  input_text: string
  result_text: string
  model: string
  instruction_key: string
  created_at?: string
  formatted_date: string
}

export interface OptionItem {
  value: string
  label: string
}

export interface ModeOption {
  value: string
  label: string
  topics?: OptionItem[]
}

export interface DirectionOption {
  value: string
  label: string
  modes: ModeOption[]
}

export interface OperationOption {
  value: string
  label: string
  directions?: DirectionOption[]
  styles?: OptionItem[]
  languages?: OptionItem[]
}

export interface TransformOptions {
  operations: OperationOption[]
}

export interface CreateInstructionPayload {
  operation: string
  direction?: string
  mode?: string
  topic?: string
  style?: string
  language?: string
  content?: string
}

export interface HistoryPage {
  items: HistoryRecord[]
  total: number
  limit: number
  offset: number
}

export interface HistoryRecord {
  id: string
  type: string
  type_display: string
  input_text: string
  result_text: string
  model: string
  instruction_key: string
  metadata?: Record<string, string>
  created_at?: string
  formatted_date: string
}

export interface QuizQuestion {
  history_id: string
  type: string
  type_display: string
  prompt: string
  answer: string
}

export interface QuizResponse {
  questions: QuizQuestion[]
}

export interface StatsBucket {
  simplify: number
  en_fa: number
  fa_en: number
  term: number
  refine: number
  symptoms: number
  compare: number
  grammar: number
  cursor: number
  frontend: number
  total: number
}

export interface StatsResponse {
  today: StatsBucket
  yesterday: StatsBucket
  week: StatsBucket
  month: StatsBucket
  all_time: StatsBucket
}

export interface Instruction {
  key: string
  content: string
  updated_at: string
}

export interface AppSettings {
  openrouter_api_key: string
  gemini_api_key: string
  opencode_go_api_key: string
  api_provider: 'openrouter' | 'gemini' | 'opencode_go'
  model_name: string
  updated_at: string
}

export interface OpenRouterModel {
  id: string
  name: string
  context_length: number
}

export interface CreditsInfo {
  source: 'credits' | 'key'
  remaining: number | null
  total_credits: number | null
  total_usage: number | null
  limit_remaining: number | null
  usage: number | null
}
