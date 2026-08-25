import request from './request'

// 配置
export interface TrimMediaConfig {
  id?: number
  host: string
  username: string
  password?: string
  access_code: string
  play_host: string
  sync_libraries: string
}

// 媒体库
export interface Library {
  id: string
  name: string
  type: string
  path: string[]
  item_count: number
  image_list: string[]
  link: string
}

// 媒体项
export interface MediaStream {
  resolutions: string[]
  audio_type: string[]
  color_range_type: string[]
}

export interface MediaItem {
  guid: string
  imdb_id: string
  trim_id: string
  tv_title: string
  parent_title: string
  title: string
  logo: string
  original_title: string
  backdrop: string
  poster: string
  poster_width: number
  poster_height: number
  vote_average: string
  genres: number[]
  release_date: string
  runtime: number
  production_countries: string[]
  overview: string
  is_favorite: number
  is_watched: number
  watched_ts: number
  air_date: string
  season_number: number
  number_of_episodes: number
  local_number_of_episodes: number
  local_number_of_seasons: number
  can_play: number
  type: string
  play_error: string
  parent_guid: string
  ancestor_guid: string
  ancestor_name: string
  ancestor_category: string
  play_item_guid: string
  duration: number
  logic_type: number
  media_stream: MediaStream
}

// 播放项
export interface PlayItem {
  id: string
  title: string
  subtitle: string
  type: string
  image: string
  link: string
  percent: number
}

// 媒体统计
export interface MediaStatistics {
  favorite: number
  movie: number
  tv: number
  video: number
  total: number
}

// 季/集
export interface Season {
  guid: string
  type: string
  tv_title: string
  parent_title: string
  title: string
  original_title: string
  overview: string
  poster: string
  backdrops: string
  posters: string
  season_number: number
  episode_number: number
}

// 演职员
export interface Person {
  item_guid: string
  person_guid: string
  role: string
  job: string
  order: number
  department: string
  trim_id: string
  imdb_id: string
  tmdb_id: number
  lan: string
  name: string
  original_name: string
  also_know_as: string
  biography: string
  known_for_department: string
  profile_path: string
  gender: number
}

// 编辑信息中的演职员条目
export interface EditCredit {
  person_guid: string
  name: string
  job: string
  role: string
  order: number
  profile_path: string
}

// 演员搜索结果
export interface PersonSearchResult {
  guid: string
  name: string
  imdbId: string
  trim_id: string
  is_official: boolean
  original_name: string
  profile: string
  is_favorite: number
}

// 编辑信息
export interface EditDetail {
  item_guid: string
  trim_id: string
  is_official: boolean
  title: string
  title_locked: boolean
  overview: string
  overview_locked: boolean
  rating: number
  rating_locked: boolean
  air_date: string
  air_date_locked: boolean
  first_air_date: string | null
  first_air_date_locked: boolean
  last_air_date: string | null
  last_air_date_locked: boolean
  content_rating: string
  content_rating_locked: boolean
  backdrops: string
  backdrops_locked: boolean
  logos: string
  logos_locked: boolean
  posters: string
  posters_locked: boolean
  poster_type: number
  genres_locked: boolean
  genres: (number | string)[]
  production_countries: string[]
  production_countries_locked: boolean
  credits: EditCredit[]
  credits_locked: boolean
  job_types_opts: string[]
  content_rating_opts: string[]
}

// --- 配置 ---

export const getConfig = () => request.get<TrimMediaConfig>('/trimmedia/config')

export const saveConfig = (data: TrimMediaConfig) => request.post('/trimmedia/config', data)

export const testConnection = (data: Partial<TrimMediaConfig>) =>
  request.post<{ status: string; version?: { frontend?: string; backend?: string }; user?: boolean }>('/trimmedia/test', data)

// --- 媒体库 ---

export const getLibraries = () => request.get<Library[]>('/trimmedia/libraries')

// --- 媒体条目 ---

export const getItems = (libraryId: string, start = 0, limit = 20) =>
  request.get<{ total: number; items: MediaItem[] }>(`/trimmedia/items/${libraryId}`, { params: { start, limit } })

export const getItem = (itemId: string) =>
  request.get<MediaItem>(`/trimmedia/item/${itemId}`)

// --- 季集 ---

export const getSeasons = (tvId: string) =>
  request.get<Season[]>(`/trimmedia/seasons/${tvId}`)

export const getEpisodes = (seasonId: string) =>
  request.get<Season[]>(`/trimmedia/episodes/${seasonId}`)

// --- 演职员 ---

export const getPersons = (itemId: string) =>
  request.get<Person[]>(`/trimmedia/persons/${itemId}`)

// 搜索演员
export const searchPersons = (keyword: string, page = 1, pageSize = 200) =>
  request.post<PersonSearchResult[]>('/trimmedia/persons/search', { keyword, page, page_size: pageSize })

// 导入演员（通过 MetaTube 搜索 → 下载图片 → 上传 → 创建）
export interface ImportPersonResult {
  guid: string
  name: string
  profile_path: string
}

export const importPerson = (name: string) =>
  request.post<ImportPersonResult>('/trimmedia/persons/import', { name })

// 下载 http 网络图片并上传到飞牛，返回飞牛图片路径
// type: 图片类型（如 poster、backdrop），默认 poster
export const downloadAndUploadImage = (url: string, type = 'poster') =>
  request.post<{ path: string }>('/trimmedia/image/download-upload', { url, type })

// --- 类型 ---

export interface Genre {
  id: number
  value: string
}

export const getGenres = (lan = 'zh-CN') =>
  request.get<Genre[]>('/trimmedia/genres', { params: { lan } })

// 批量新增自定义分类
export const batchCreateGenres = (values: string[]) =>
  request.post<Genre[]>('/trimmedia/genres/batch', { values })

// --- 国家地区 ---

export interface Country {
  key: string
  value: string
}

export const getCountries = (lan = 'zh-CN') =>
  request.get<Country[]>('/trimmedia/countries', { params: { lan } })

// --- 编辑信息 ---

export const getEditDetail = (itemId: string) =>
  request.get<EditDetail>(`/trimmedia/edit/${itemId}`)

export const saveEditDetail = (itemId: string, data: EditDetail) =>
  request.post<{ success: boolean }>(`/trimmedia/edit/${itemId}`, data)

// --- 播放 ---

export const getPlayURL = (itemId: string) =>
  request.get<{ url: string }>(`/trimmedia/playurl/${itemId}`)

export const getResume = (num = 12) =>
  request.get<PlayItem[]>('/trimmedia/resume', { params: { num } })

export const getLatest = (num = 20) =>
  request.get<PlayItem[]>('/trimmedia/latest', { params: { num } })

// --- 统计 ---

export const getStatistics = () =>
  request.get<MediaStatistics>('/trimmedia/statistics')

// --- 刷新 ---

export const refreshLibraries = (paths?: string[]) =>
  request.post<{ success: boolean }>('/trimmedia/refresh', { paths: paths || [] })

// --- 搜索 ---

export const searchMedia = (q: string) =>
  request.get<MediaItem[]>('/trimmedia/search', { params: { q } })
