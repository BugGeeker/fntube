import request from './request'

export interface MovieSearchSource {
  name: string
  label: string
}

export interface MovieSearchConfig {
  source: string
  javdb_domain: string
  javbus_domain: string
}

export interface MovieSearchResult {
  source: string
  hash: string
  name: string
  size: string
  date: string
  files?: number
  tags?: string[]
  magnet: string
  cover_url?: string
  number?: string
  categories?: string[]
  title?: string
  maker?: string
  release_date?: string
  runtime?: number
  series?: string
  label?: string
  director?: string
  actors?: string[]
}

export const getMovieSearchSources = () =>
  request.get<{ data: MovieSearchSource[] }>('/movie-search/sources')

export const getMovieSearchConfig = () =>
  request.get<MovieSearchConfig>('/movie-search/config')

export const saveMovieSearchConfig = (config: MovieSearchConfig) =>
  request.post<{ status: string; config: MovieSearchConfig }>('/movie-search/config', config)

export const searchMovies = (source: string, q: string) =>
  request.get<{ data: MovieSearchResult[] }>('/movie-search/search', {
    params: { source, q },
  })
