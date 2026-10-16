import request from './request'

export interface StorageMapping {
  id: number
  name: string
  guangya_folder_id: string
  guangya_folder_path: string
  smartstrm_task_name: string
  strm_save_path: string
  created_at?: string
  updated_at?: string
}

export type StorageMappingInput = Omit<StorageMapping, 'id' | 'created_at' | 'updated_at'>

export const getStorageMappings = () =>
  request.get<StorageMapping[]>('/storage-config/mappings')

export const createStorageMapping = (mapping: StorageMappingInput) =>
  request.post<StorageMapping>('/storage-config/mappings', mapping)

export const updateStorageMapping = (id: number, mapping: StorageMappingInput) =>
  request.put<StorageMapping>(`/storage-config/mappings/${id}`, mapping)

export const deleteStorageMapping = (id: number) =>
  request.delete<{ status: string }>(`/storage-config/mappings/${id}`)
