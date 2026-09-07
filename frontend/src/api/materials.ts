import { apiClient, unwrap } from './client'
import type {
  MaterialChunk,
  MaterialDetail,
  MaterialDownload,
  MaterialIndexStatus,
  MaterialList,
  MaterialUploadResult,
  GenerationJob,
} from '@/types/material'

export const materialsApi = {
  list: (spaceId: string) =>
    unwrap<MaterialList>(apiClient.get(`/spaces/${spaceId}/materials`)),
  upload: (spaceId: string, file: File, materialKind:'study'|'past_exam'='study', onProgress?: (percentage: number) => void) => {
    const body = new FormData()
    body.append('file', file)
    body.append('material_kind', materialKind)
    return unwrap<MaterialUploadResult>(
      apiClient.post(`/spaces/${spaceId}/materials`, body, {
        timeout: 120_000,
        onUploadProgress: (event) => {
          if (event.total) onProgress?.(Math.round((event.loaded / event.total) * 100))
        },
      }),
    )
  },
  getJob: (spaceId: string, jobId: string) =>
    unwrap<GenerationJob>(apiClient.get(`/spaces/${spaceId}/jobs/${jobId}`)),
  listChunks: (spaceId: string, materialId: string) =>
    unwrap<{ items: MaterialChunk[] }>(
      apiClient.get(`/spaces/${spaceId}/materials/${materialId}/chunks`),
    ),
  getIndexStatus: (spaceId: string, materialId: string) =>
    unwrap<MaterialIndexStatus>(
      apiClient.get(`/spaces/${spaceId}/materials/${materialId}/index-status`),
    ),
  get: (spaceId: string, materialId: string) =>
    unwrap<MaterialDetail>(apiClient.get(`/spaces/${spaceId}/materials/${materialId}`)),
  retry: (spaceId: string, materialId: string) =>
    unwrap<MaterialUploadResult>(
      apiClient.post(`/spaces/${spaceId}/materials/${materialId}/retry`),
    ),
  remove: (spaceId: string, materialId: string) =>
    apiClient.delete(`/spaces/${spaceId}/materials/${materialId}`),
  getDownload: (spaceId: string, materialId: string) =>
    unwrap<MaterialDownload>(
      apiClient.get(`/spaces/${spaceId}/materials/${materialId}/download-url`),
    ),
}
