import request from './request'
import type { ScrapeLog } from './scrapelog'

// 刮削任务运行记录
export interface TaskRunRecord {
  id: number
  task_id: number
  task_name: string
  library_name: string
  start_time: string
  end_time: string | null
  duration: number // 运行时长（秒）
  success_count: number
  completed_count: number
  failed_count: number
  status: string // running / done / error
  error: string
  created_at: string
  updated_at: string
}

// 运行记录明细
export interface TaskRunDetail {
  record: TaskRunRecord
  logs: ScrapeLog[]
}

// 获取运行记录列表（分页）
export const getTaskRunRecords = (page = 1, pageSize = 20) =>
  request.get<{ total: number; items: TaskRunRecord[] }>('/taskrun/list', { params: { page, page_size: pageSize } })

// 获取运行记录明细
export const getTaskRunDetail = (id: number) =>
  request.get<TaskRunDetail>(`/taskrun/detail/${id}`)
