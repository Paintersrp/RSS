import ky from 'ky'

const baseURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export const api = ky.create({
  prefixUrl: baseURL,
  headers: {
    'Content-Type': 'application/json',
  },
})

export type Feed = {
  id: string
  url: string
  title: string
  etag?: string
  last_modified?: string
  last_crawled?: string
}

export type Item = {
  id: string
  feed_id: string
  feed_title: string
  guid?: string
  url: string
  title: string
  author?: string
  content_html: string
  content_text: string
  published_at?: string
  retrieved_at: string
}

export type SearchHit = {
  id: string
  feed_id: string
  feed_title?: string
  url?: string
  title: string
  content_text: string
  published_at?: string
}

export async function listFeeds() {
  return api.get('feeds').json<Feed[]>()
}

export async function listItems(params: { limit?: number; offset?: number } = {}) {
  const searchParams = new URLSearchParams()
  if (params.limit) searchParams.set('limit', String(params.limit))
  if (params.offset) searchParams.set('offset', String(params.offset))
  return api.get('items', { searchParams }).json<Item[]>()
}

export async function searchItems(params: { q: string; limit?: number; offset?: number }) {
  const searchParams = new URLSearchParams()
  searchParams.set('q', params.q)
  if (params.limit) searchParams.set('limit', String(params.limit))
  if (params.offset) searchParams.set('offset', String(params.offset))
  return api.get('search', { searchParams }).json<{ hits: SearchHit[] }>()
}

export async function addFeed(url: string) {
  return api.post('feeds', { json: { url } }).json<Feed>()
}
