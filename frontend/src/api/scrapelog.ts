import request from './request'

// 刮削步骤记录
export interface ScrapeStep {
  step: string
  status: string // running | success | failed
  error?: string
}

// 刮削日志
export interface ScrapeLog {
  id: number
  item_guid: string
  title: string
  number: string
  method: string // manual | auto
  status: string // in_progress | success | failed | completed
  steps: string // JSON 字符串，解析为 ScrapeStep[]
  error: string
  created_at: string
  updated_at: string
  current_step: string
  progress: number
}

// 获取刮削日志列表（分页）
export const getScrapeLogs = (page = 1, pageSize = 20, number = '', status = '') =>
  request.get<{ total: number; items: ScrapeLog[] }>('/scrapelog/list', {
    params: {
      page,
      page_size: pageSize,
      number: number || undefined,
      status: status || undefined,
    },
  })

// 创建刮削日志
export const createScrapeLog = (data: { item_guid: string; title: string; number: string; method: string }) =>
  request.post<ScrapeLog>('/scrapelog/create', data)

// 删除刮削日志
export const deleteScrapeLog = (id: number) =>
  request.delete(`/scrapelog/${id}`)

// 重新刮削指定媒体项
export const rescrapeItem = (itemGuid: string) =>
  request.post<{ status: string; message: string }>(`/scrapelog/rescrape/${itemGuid}`)

export const scrapeLogEventsURL = './api/scrapelog/events'
