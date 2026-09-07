import { apiClient, unwrap } from './client'
import type { CursorPage } from '@/types/api'
import type { CreateSpaceInput, LearningSpace, SpaceDetail } from '@/types/learning-space'

export const learningSpacesApi = {
  list: () => unwrap<CursorPage<LearningSpace>>(apiClient.get('/spaces')),
  get: (spaceId: string) => unwrap<SpaceDetail>(apiClient.get(`/spaces/${spaceId}`)),
  create: (input: CreateSpaceInput) =>
    unwrap<LearningSpace>(
      apiClient.post('/spaces', input, { headers: { 'Idempotency-Key': crypto.randomUUID() } }),
    ),
  remove: (spaceId: string) => apiClient.delete(`/spaces/${spaceId}`),
}
