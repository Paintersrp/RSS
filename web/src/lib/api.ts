import ky from 'ky'

export interface Item {
  id: string
  feed_id: string
  feed_title: string
  guid?: string | null
  url: string
  title: string
  author?: string | null
  content_html: string
  content_text: string
  published_at?: string | null
  retrieved_at: string
}

export interface HealthResponse {
  status: string
}

export interface SearchDocument {
  id: string
  feed_id: string
  feed_title: string
  title: string
  content_text: string
  url: string
  published_at?: string | null
}

export interface SearchResponse {
  query: string
  limit: number
  offset: number
  estimated_total: number
  hits: SearchDocument[]
}

const baseUrl = (
  (import.meta.env.VITE_API_URL as string | undefined)?.replace(/\/$/, '') ??
  'http://localhost:8080'
)

export const api = ky.create({
  prefixUrl: baseUrl,
  headers: {
    Accept: 'application/json',
  },
  retry: 0,
})

export interface ListItemsParams {
  limit?: number
  offset?: number
}

export async function listRecentItems({
  limit = 50,
  offset = 0,
}: ListItemsParams = {}): Promise<Item[]> {
  const searchParams = new URLSearchParams()
  if (limit) {
    searchParams.set('limit', String(limit))
  }
  if (offset) {
    searchParams.set('offset', String(offset))
  }
  return api.get('items', { searchParams }).json<Item[]>()
}

export interface SearchItemsParams {
  query: string
  limit?: number
  offset?: number
}

export async function searchItems({
  query,
  limit = 20,
  offset = 0,
}: SearchItemsParams): Promise<SearchResponse> {
  const searchParams = new URLSearchParams()
  searchParams.set('q', query)
  if (limit) {
    searchParams.set('limit', String(limit))
  }
  if (offset) {
    searchParams.set('offset', String(offset))
  }
  return api.get('search', { searchParams }).json<SearchResponse>()
}

export async function getHealth(): Promise<HealthResponse> {
  return api.get('healthz').json<HealthResponse>()
}
