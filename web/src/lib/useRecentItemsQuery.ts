import { useQuery, type UseQueryOptions, type UseQueryResult } from '@tanstack/react-query'

import {
  listRecentItems,
  type ListItemsParams,
  type ListRecentItemsResponse,
} from '@/lib/api'
import { queryKeys } from '@/lib/query'

export type RecentItemsSortField = NonNullable<ListItemsParams['sortField']>
export type RecentItemsSortDirection = NonNullable<ListItemsParams['sortDirection']>
export type RecentItemsSort = `${RecentItemsSortField}:${RecentItemsSortDirection}`

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
  const [sortFieldRaw, sortDirectionRaw] = sort.split(':') as [
    RecentItemsSortField?,
    RecentItemsSortDirection?,
  ]
  const sortField: RecentItemsSortField =
    sortFieldRaw === 'retrieved_at' ? 'retrieved_at' : 'published_at'
  const sortDirection: RecentItemsSortDirection =
    sortDirectionRaw === 'asc' ? 'asc' : 'desc'

  return {
    limit,
    offset,
    sortField,
    sortDirection,
    feed_ids: sortedFeedIds,
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
