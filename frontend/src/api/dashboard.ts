import request from './request'

// 总览页每日刮削趋势
export interface DashboardDailyTrend {
  date: string
  count: number
}

// 总览页刮削任务状态
export interface DashboardTaskStatus {
  id: number
  name: string
  library_name: string
  enabled: boolean
  is_running: boolean
  interval: number
  last_run_at: string | null
}

// 总览页汇总数据
export interface DashboardSummary {
  total_media: number
  weekly_new_media: number
  total_scrapes: number
  weekly_scrapes: number
  daily_scrapes: DashboardDailyTrend[]
  task_summary: DashboardTaskStatus[]
}

export const getDashboardSummary = () =>
  request.get<DashboardSummary>('/dashboard/summary')
