export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type RecordType = 'invites' | 'rebates' | 'transfers'
