import request from './request'

export interface SmartStrmConfig {
  id?: number
  url: string
  username: string
  password: string
  strm_root_path: string
}

export interface SmartStrmVersion {
  status: string
  version: string
  remote_version: string
  is_new: boolean
  notify: string
  license_store_url: string | null
  change_log_url: string
}

export interface SmartStrmTask {
  name: string
  config: Record<string, unknown>
  next_run: unknown
  running: boolean
}

export const getSmartStrmConfig = () => request.get<SmartStrmConfig>('/smartstrm/config')

export const saveSmartStrmConfig = (config: SmartStrmConfig) =>
  request.post<{ status: string; config: SmartStrmConfig }>('/smartstrm/config', config)

export const testSmartStrmConnection = (config: Pick<SmartStrmConfig, 'url' | 'username' | 'password'>) =>
  request.post<SmartStrmVersion>('/smartstrm/test', config)

export const getSmartStrmTasks = () =>
  request.get<Record<string, SmartStrmTask>>('/smartstrm/tasks')
