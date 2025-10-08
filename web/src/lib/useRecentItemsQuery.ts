import { useQuery, type UseQueryOptions, type UseQueryResult } from '@tanstack/react-query'

import { listRecentItems, type ListRecentItemsResponse } from '@/lib/api'
import { queryKeys } from '@/lib/query'

export type RecentItemsSort = 'published_at:desc' | 'published_at:asc'

export interface RecentItemsQueryState {
  feeds: string[]
  page: number
  limit: number
  sort: RecentItemsSort
}

export type RecentItemsQueryOptions = Pick<
  UseQueryOptions<ListRecentItemsResponse, unknown, ListRecentItemsResponse>,
  'enabled' | 'keepPreviousData'
>

export function buildListParams({
  feeds,
  page,
  limit,
  sort,
}: RecentItemsQueryState) {
  const safePage = Number.isFinite(page) && page > 0 ? page : 1
  const offset = (safePage - 1) * limit
  const feedIds = [...feeds].filter((id) => typeof id === 'string' && id.length > 0)
  const sortedFeedIds = feedIds.length > 0 ? [...new Set(feedIds)].sort() : undefined

  return {
    limit,
    offset,
    sort,
    feed_id: sortedFeedIds,
  } as const
}

export function buildQueryKey(state: RecentItemsQueryState) {
  return queryKeys.recentItems(buildListParams(state))
}

export function useRecentItemsQuery(
  state: RecentItemsQueryState,
  options: RecentItemsQueryOptions = {},
): UseQueryResult<ListRecentItemsResponse> {
  const params = buildListParams(state)
  return useQuery({
    queryKey: queryKeys.recentItems(params),
    queryFn: () => listRecentItems(params),
    ...options,
  })
}
