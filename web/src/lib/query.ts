import { QueryClient } from '@tanstack/react-query'

import type { ListItemsParams, SearchItemsParams } from './api'

export const queryKeys = {
  health: () => ['health'] as const,
  recentItems: (params: ListItemsParams = {}) => ['items', params] as const,
  search: (params: SearchItemsParams) => ['search', params] as const,
}

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: 1,
        staleTime: 30_000,
      },
    },
  })
}
