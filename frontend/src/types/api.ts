export interface ApiEnvelope<T> {
  code: string
  message: string
  data: T
}

export interface ApiErrorDetail {
  field?: string
  reason: string
}

export interface ApiErrorBody {
  code: string
  message: string
  details?: ApiErrorDetail[]
  request_id?: string
}

export interface CursorPage<T> {
  items: T[]
  next_cursor: string | null
}
