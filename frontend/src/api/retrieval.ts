import { apiClient, unwrap } from './client'
import type { RetrievalSearchResult } from '@/types/retrieval'

export const retrievalApi = {
  search: (spaceId: string, query: string, materialIds?: string[]) =>
    unwrap<RetrievalSearchResult>(
      apiClient.post(`/spaces/${spaceId}/retrieval/search`, {
        query,
        top_k: 8,
        material_ids: materialIds ?? [],
      }),
    ),
}
