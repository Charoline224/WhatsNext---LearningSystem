export type MaterialStatus = 'queued' | 'processing' | 'ready' | 'failed'
export type JobStatus = 'queued' | 'processing' | 'succeeded' | 'failed'

export interface LearningMaterial {
  id: string
  learning_space_id: string
  material_kind: 'study' | 'past_exam'
  original_name: string
  mime_type: string
  size_bytes: number
  status: MaterialStatus
  failure_reason: string | null
  created_at: string
  updated_at: string
}

export interface GenerationJob {
  id: string
  learning_space_id: string
  material_id: string
  job_type: 'process_material' | 'embed_material'
  status: JobStatus
  progress: number
  attempts: number
  max_attempts: number
  started_at: string | null
  completed_at: string | null
  error_code: string | null
  error_message: string | null
  created_at: string
  updated_at: string
}

export interface MaterialListItem {
  material: LearningMaterial
  job: GenerationJob
  chunk_count: number
}

export interface MaterialList {
  items: MaterialListItem[]
}

export type MaterialUploadResult = MaterialListItem

export interface MaterialChunk {
  id: string
  material_id: string
  chunk_index: number
  content: string
  source_type: 'page' | 'slide' | 'document' | 'text'
  source_start: number | null
  source_end: number | null
  char_count: number
  token_estimate: number
  created_at: string
}

export type MaterialIndexState =
  | 'not_started'
  | 'pending'
  | 'indexing'
  | 'indexed'
  | 'partial'
  | 'failed'

export interface MaterialIndexStatus {
  material_id: string
  status: MaterialIndexState
  embedding_model: string
  total_chunks: number
  pending_chunks: number
  indexed_chunks: number
  failed_chunks: number
  progress: number
  job: GenerationJob | null
}

export interface MaterialDetail {
  material: LearningMaterial
  processing_job: GenerationJob
  embedding_job: GenerationJob | null
  chunk_count: number
}

export interface MaterialDownload {
  url: string
  expires_at: string
}
