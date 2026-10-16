import request from './request'

export interface CopyTestResult {
  source: string
  destination: string
  files: number
  directories: number
  bytes: number
}

export interface ACLResult {
  path: string
  readable: boolean
  writable: boolean
  deletable: boolean
}

export function testCopy(source: string, destination: string) {
  return request.post<CopyTestResult>('/system-test/copy', { source, destination })
}

export function getSharedFolders() {
  return request.get<{ paths: string[] }>('/system-test/shared-folders')
}

export function checkACL(uid: number, path: string[]) {
  return request.post<ACLResult[]>('/system-test/acl', { uid, path })
}
