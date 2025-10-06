import { createRouter } from '@tanstack/react-router'
import type { QueryClient } from '@tanstack/react-query'

import { Route as RootRoute } from './routes/__root'
import { Route as IndexRoute } from './routes/index'
import { Route as SearchRoute } from './routes/search'

const routeTree = RootRoute.addChildren([IndexRoute, SearchRoute])

export interface RouterContext {
  queryClient: QueryClient
}

export const router = createRouter({
  routeTree,
  context: {
    queryClient: undefined!,
  },
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

export function createAppRouter(ctx: RouterContext) {
  return router.withContext(ctx)
}
