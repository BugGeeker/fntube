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