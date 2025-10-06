import * as React from 'react'
import { Link, Outlet, createRootRouteWithContext, useRouter, useRouterState } from '@tanstack/react-router'
import { RouterDevtools } from '@tanstack/router-devtools'
import { useQuery } from '@tanstack/react-query'
import { Circle, Rss, Search } from 'lucide-react'

import { listFeeds } from '../lib/api'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Toaster } from '../components/ui/toaster'
import type { RouterContext } from '../router'

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootComponent,
})

function RootComponent() {
  const router = useRouter()
  const { pending } = useRouterState({ select: (s) => ({ pending: s.status === 'pending' }) })
  const { data: feeds } = useQuery({ queryKey: ['feeds'], queryFn: listFeeds })
  const [query, setQuery] = React.useState('')

  const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    router.navigate({ to: '/search', search: { q: query } })
  }

  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b bg-background">
        <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-6 py-4">
          <Link to="/" className="flex items-center gap-2 text-xl font-semibold">
            <Rss className="h-5 w-5" /> Courier
          </Link>
          <form onSubmit={onSubmit} className="flex max-w-lg flex-1 items-center gap-2">
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search articles..."
            />
            <Button type="submit" variant="outline" disabled={!query}>
              <Search className="h-4 w-4" />
            </Button>
          </form>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Circle className="h-3 w-3 text-green-500" />
            {pending ? 'Loading…' : 'Live'}
          </div>
        </div>
      </header>
      <main className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-6 py-6">
        <Outlet />
      </main>
      <footer className="border-t bg-muted py-4 text-center text-xs text-muted-foreground">
        Tracking {feeds?.length ?? 0} feeds • courier v0.1
      </footer>
      {import.meta.env.DEV ? <RouterDevtools position="bottom-right" /> : null}
      <Toaster />
    </div>
  )
}
