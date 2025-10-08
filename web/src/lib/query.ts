import { QueryCache, QueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import type { ListItemsParams, SearchItemsParams } from './api'

type QueryMeta = {
  skipGlobalErrorToast?: boolean
}

export const queryKeys = {
  health: () => ['health'] as const,
  recentItems: (params: ListItemsParams) => ['items', params] as const,
  search: (params: SearchItemsParams) => ['search', params] as const,
  feeds: () => ['feeds'] as const,
}

export function createQueryClient() {
  const queryCache = new QueryCache({
    onError: (error, query) => {
      const meta = query?.meta as QueryMeta | undefined

      if (meta?.skipGlobalErrorToast) {
        return
      }

      const message =
        error instanceof Error ? error.message : 'Please try again later.'
      toast.error('Request failed', {
        description: message,
      })
    },
  })

  return new QueryClient({
    queryCache,
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: 1,
        staleTime: 30_000,
      },
    },
  })
}
