import request from './request'

export interface GuangyaConfig {
  bound: boolean
  subject?: string
  name?: string
  phone_number?: string
  login_method?: 'qr' | 'sms'
  token_expires_at?: string
  updated_at?: string
}

export interface GuangyaQRSession {
  session_id: string
  user_code: string
  verification_url: string
  verification_uri_complete: string
  interval: number
  expires_at: string
}

export interface GuangyaQRPollResult {
  state: 'pending' | 'bound' | 'expired' | 'denied'
  config?: GuangyaConfig
}

export interface GuangyaSMSSendResult {
  session_id?: string
  captcha_required: boolean
  captcha_url?: string
}

export interface GuangyaCloudFile {
  fileName: string
  fileSize?: number
  fileIndex?: number
  isDir?: boolean
  subfiles?: GuangyaCloudFile[]
}

export interface GuangyaCloudResolved {
  resType: number
  url?: string
  infoHash?: string
  fileName?: string
  fileSize?: number
  subfiles?: GuangyaCloudFile[]
}

export interface GuangyaCloudFolder {
  fileId: string
  fileName: string
  fileSize?: number
  resType: number
  parentId?: string
}

export interface GuangyaCloudFolderList {
  list: GuangyaCloudFolder[]
  total: number
}

export interface GuangyaCloudTask {
  taskId: string
  url: string
}

export interface GuangyaCloudAddJob {
  id: number
  task_id: string
  parent_id: string
  requested_name: string
  resolved_is_directory: boolean
  status: 'pending' | 'completed' | 'failed'
  current_step: string
  progress: number
  error: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface GuangyaCloudAddJobPage {
  total: number
  items: GuangyaCloudAddJob[]
}

export interface GuangyaSelectedCloudFile {
  fileIndex: number
  fileName: string
  fileSize?: number
  path?: string[]
}

export const getGuangyaConfig = () =>
  request.get<GuangyaConfig>('/guangya/config')

export const unbindGuangya = () =>
  request.post('/guangya/unbind')

export const startGuangyaQR = () =>
  request.post<GuangyaQRSession>('/guangya/qr/start')

export const pollGuangyaQR = (sessionId: string) =>
  request.post<GuangyaQRPollResult>('/guangya/qr/poll', { session_id: sessionId })

export const sendGuangyaSMS = (phone: string, sessionId = '', captchaToken = '') =>
  request.post<GuangyaSMSSendResult>('/guangya/sms/send', {
    phone,
    session_id: sessionId,
    captcha_token: captchaToken,
  })

export const verifyGuangyaSMS = (sessionId: string, code: string) =>
  request.post<GuangyaConfig>('/guangya/sms/verify', {
    session_id: sessionId,
    code,
  })

export const resolveGuangyaCloudURL = (url: string) =>
  request.post<GuangyaCloudResolved>('/guangya/cloud/resolve', { url })

export const listGuangyaCloudFolders = (parentId = '', page = 0, pageSize = 50) =>
  request.get<GuangyaCloudFolderList>('/guangya/cloud/folders', {
    params: { parent_id: parentId, page, page_size: pageSize },
  })

export const createGuangyaCloudFolder = (name: string, parentId = '') =>
  request.post<GuangyaCloudFolder>('/guangya/cloud/folders', {
    name,
    parent_id: parentId,
  })

export const createGuangyaCloudTask = (payload: {
  url: string
  parentId: string
  newName: string
  fileIndexes: number[]
  resolvedIsDirectory: boolean
  selectedFiles: GuangyaSelectedCloudFile[]
}) =>
  request.post<GuangyaCloudTask>('/guangya/cloud/task', {
    url: payload.url,
    parent_id: payload.parentId,
    new_name: payload.newName,
    file_indexes: payload.fileIndexes,
    resolved_is_directory: payload.resolvedIsDirectory,
    selected_files: payload.selectedFiles,
  })

export const getGuangyaCloudAddJobs = (page = 1, pageSize = 20) =>
  request.get<GuangyaCloudAddJobPage>('/guangya/cloud/jobs', {
    params: { page, page_size: pageSize },
  })

export const deleteGuangyaCloudAddJob = (id: number) =>
  request.delete(`/guangya/cloud/jobs/${id}`)

export const guangyaCloudAddEventsURL = './api/guangya/cloud/jobs/events'
