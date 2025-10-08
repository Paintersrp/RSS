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

export interface Feed {
  id: string
  url: string
  title: string
  last_crawled?: string | null
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
  sort?: string
  feed_id?: string | string[]
}

export interface ListRecentItemsResponse {
  items: Item[]
  total: number
}

export async function listRecentItems({
  limit = 50,
  offset = 0,
  sort,
  feed_id,
}: ListItemsParams = {}): Promise<ListRecentItemsResponse> {
  const searchParams = new URLSearchParams()
  if (limit) {
    searchParams.set('limit', String(limit))
  }
  if (offset) {
    searchParams.set('offset', String(offset))
  }
  if (sort) {
    searchParams.set('sort', sort)
  }
  if (Array.isArray(feed_id)) {
    for (const id of feed_id) {
      if (id) {
        searchParams.append('feed_id', id)
      }
    }
  } else if (feed_id) {
    searchParams.set('feed_id', feed_id)
  }

  const response = await api.get('items', { searchParams })
  const payload = await response.json<
    Item[] | { items?: Item[]; total?: number }
  >()

  const headerTotal = response.headers.get('x-total-count')
  const items = Array.isArray(payload) ? payload : payload.items ?? []
  const totalFromPayload = !Array.isArray(payload) && payload.total

  const total =
    typeof totalFromPayload === 'number'
      ? totalFromPayload
      : headerTotal
        ? Number.parseInt(headerTotal, 10)
        : items.length

  return { items, total }
}

export interface SearchItemsParams {
  query: string
  limit?: number
  offset?: number
  feed_id?: string
}

export async function searchItems({
  query,
  limit = 20,
  offset = 0,
  feed_id,
}: SearchItemsParams): Promise<SearchResponse> {
  const searchParams = new URLSearchParams()
  searchParams.set('q', query)
  if (limit) {
    searchParams.set('limit', String(limit))
  }
  if (offset) {
    searchParams.set('offset', String(offset))
  }
  if (feed_id) {
    searchParams.set('feed_id', feed_id)
  }
  return api.get('search', { searchParams }).json<SearchResponse>()
}

export async function getHealth(): Promise<HealthResponse> {
  return api.get('healthz').json<HealthResponse>()
}

export async function listFeeds(): Promise<Feed[]> {
  return api.get('feeds').json<Feed[]>()
}
