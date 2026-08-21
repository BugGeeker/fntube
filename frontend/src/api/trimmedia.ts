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

export interface MediaServerItem {
  guid: string
  imdb_id: string
  trim_id: string
  tv_title: string
  parent_title: string
  title: string
  logos: string
  original_title: string
  backdrops: string
  posters: string
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
export interface SeasonItem {
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

// --- 配置 ---

export const getConfig = () => request.get<TrimMediaConfig>('/trimmedia/config')

export const saveConfig = (data: TrimMediaConfig) => request.post('/trimmedia/config', data)

export const testConnection = (data: Partial<TrimMediaConfig>) =>
  request.post<{ status: string; version?: { frontend?: string; backend?: string }; user?: boolean }>('/trimmedia/test', data)

// --- 媒体库 ---

export const getLibraries = () => request.get<Library[]>('/trimmedia/libraries')

// --- 媒体条目 ---

export const getItems = (libraryId: string, start = 0, limit = 20) =>
  request.get<MediaServerItem[]>(`/trimmedia/items/${libraryId}`, { params: { start, limit } })

export const getItem = (itemId: string) =>
  request.get<MediaServerItem>(`/trimmedia/item/${itemId}`)

// --- 季集 ---

export const getSeasons = (tvId: string) =>
  request.get<SeasonItem[]>(`/trimmedia/seasons/${tvId}`)

export const getEpisodes = (seasonId: string) =>
  request.get<SeasonItem[]>(`/trimmedia/episodes/${seasonId}`)

// --- 演职员 ---

export const getPersons = (itemId: string) =>
  request.get<Person[]>(`/trimmedia/persons/${itemId}`)

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
  request.get<MediaServerItem[]>('/trimmedia/search', { params: { q } })
