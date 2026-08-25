import request from './request'

// 配置
export interface MetaTubeConfig {
  id?: number
  host: string
  token: string
  translate_mode: string
  translate_engine: string
  engine_config: string
}

// 引擎专属配置
export interface BaiduEngineConfig {
  app_id: string
  secret_key: string
}

export interface DeepLEngineConfig {
  api_key: string
}

export interface GoogleEngineConfig {
  api_key: string
}

export interface OpenAIEngineConfig {
  api_key: string
  model: string
  base_url: string
}

// 翻译模式选项
export const TranslateModeOptions = [
  { value: 'none', label: '不翻译' },
  { value: 'title', label: '仅标题' },
  { value: 'summary', label: '仅简介' },
  { value: 'title_and_summary', label: '标题和简介' },
]

// 翻译引擎选项
export const TranslateEngineOptions = [
  { value: 'baidu', label: '百度翻译' },
  { value: 'deepl', label: 'DeepL' },
  { value: 'google', label: 'Google 翻译' },
  { value: 'googlefree', label: 'Google Free' },
  { value: 'openai', label: 'OpenAI' },
]

// --- API ---

export const getConfig = () => request.get<MetaTubeConfig>('/metatube/config')

export const saveConfig = (data: MetaTubeConfig) =>
  request.post('/metatube/config', data)

export const testConnection = (data: Partial<MetaTubeConfig>) =>
  request.post<{ status: string; app?: string; version?: string; token_valid?: boolean }>('/metatube/test', data)

// 影片搜索结果
export interface MovieSearchResult {
  id: string
  number: string
  title: string
  provider: string
  homepage: string
  thumb_url: string
  cover_url: string
  score: number
  actors?: string[]
  release_date: string
}

// 搜索影片
export const searchMovies = (q: string) =>
  request.get<{ data: MovieSearchResult[] }>('/metatube/search', { params: { q } })

// 影片详情
export interface MovieInfo {
  id: string
  number: string
  title: string
  summary: string
  provider: string
  homepage: string
  director: string
  actors: string[]
  thumb_url: string
  big_thumb_url: string
  cover_url: string
  big_cover_url: string
  preview_video_url: string
  preview_video_hls_url: string
  preview_images: string[]
  maker: string
  label: string
  series: string
  genres: string[]
  score: number
  runtime: number
  release_date: string
}

// 获取影片详情
export const getMovieInfo = (provider: string, id: string) =>
  request.get<{ data: MovieInfo }>(`/metatube/movie/${provider}/${id}`)

// 翻译结果
export interface TranslateResult {
  from: string
  to: string
  translated_text: string
}

// 文本翻译
export const translateText = (q: string, to?: string) =>
  request.get<{ data: TranslateResult }>('/metatube/translate', { params: { q, to } })