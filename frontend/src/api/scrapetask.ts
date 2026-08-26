import request from './request'

// 刮削计划任务
export interface ScrapeTask {
  id: number
  name: string
  library_id: string
  library_name: string
  interval: number // 扫描频率（分钟）
  enabled: boolean
  last_run_at: string | null
  is_running?: boolean // 是否正在运行
  created_at: string
  updated_at: string
}

// 获取刮削计划任务列表
export const getScrapeTasks = () =>
  request.get<ScrapeTask[]>('/scrapetask/list')

// 创建刮削计划任务
export const createScrapeTask = (data: { name: string; library_id: string; library_name: string; interval: number; enabled: boolean }) =>
  request.post<ScrapeTask>('/scrapetask/create', data)

// 更新刮削计划任务
export const updateScrapeTask = (data: { id: number; name: string; library_id: string; library_name: string; interval: number; enabled: boolean }) =>
  request.post<ScrapeTask>('/scrapetask/update', data)

// 删除刮削计划任务
export const deleteScrapeTask = (id: number) =>
  request.delete(`/scrapetask/${id}`)

// 立即执行刮削计划任务
export const runScrapeTask = (id: number) =>
  request.post<{ status: string; message: string }>(`/scrapetask/run/${id}`)
