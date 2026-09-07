export interface RetrievalResult {
  chunk_id: string
  material_id: string
  material_name: string
  chunk_index: number
  content: string
  source_type: string
  source_start: number | null
  source_end: number | null
  score: number
}

export interface RetrievalSearchResult {
  items: RetrievalResult[]
}
