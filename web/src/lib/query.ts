import { QueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import type { ListItemsParams, SearchItemsParams } from './api'

export const queryKeys = {
  health: () => ['health'] as const,
  recentItems: (params: ListItemsParams = {}) => ['items', params] as const,
  search: (params: SearchItemsParams) => ['search', params] as const,
  feeds: () => ['feeds'] as const,
}

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: 1,
        staleTime: 30_000,
        onError: (error) => {
          const message =
            error instanceof Error ? error.message : 'Please try again later.'
          toast.error('Request failed', {
            description: message,
          })
        },
      },
    },
  })
}
