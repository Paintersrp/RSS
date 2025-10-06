import { Link, useRouter, useRouterState } from '@tanstack/react-router'
import { Search } from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'

import HealthIndicator from './HealthIndicator'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { cn } from '@/lib/utils'

export default function Header() {
  const router = useRouter()
  const { location } = useRouterState()
  const currentSearch = (location.search ?? {}) as { q?: unknown }
  const [term, setTerm] = useState(() =>
    typeof currentSearch.q === 'string' ? currentSearch.q : '',
  )

  useEffect(() => {
    if (typeof currentSearch.q === 'string') {
      setTerm(currentSearch.q)
    } else if (!currentSearch.q) {
      setTerm('')
    }
  }, [currentSearch.q])

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const query = term.trim()
    void router.navigate({
      to: '/search',
      search: { q: query, page: 1 },
    })
  }

  return (
    <header className="sticky top-0 z-30 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex max-w-6xl flex-col gap-4 px-4 py-4 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center justify-between gap-6">
          <Link
            to="/"
            className="inline-flex items-center gap-3 text-lg font-semibold text-foreground transition-colors hover:text-primary"
          >
            <span className="inline-flex size-9 items-center justify-center rounded-md bg-primary text-base font-bold text-primary-foreground">
              RSS
            </span>
            <span>Courier</span>
          </Link>
          <nav className="hidden items-center gap-4 text-sm font-medium md:flex">
            <Link
              to="/"
              className={({ isActive }) =>
                cn(
                  'transition-colors text-muted-foreground hover:text-foreground',
                  isActive && 'text-foreground',
                )
              }
            >
              Recent
            </Link>
            <Link
              to="/search"
              className={({ isActive }) =>
                cn(
                  'transition-colors text-muted-foreground hover:text-foreground',
                  isActive && 'text-foreground',
                )
              }
            >
              Search
            </Link>
          </nav>
        </div>
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-end">
          <form
            onSubmit={handleSubmit}
            className="flex w-full items-center gap-2 md:w-96"
            role="search"
          >
            <Input
              placeholder="Search feeds, titles, and content…"
              value={term}
              onChange={(event) => setTerm(event.target.value)}
              aria-label="Search items"
            />
            <Button type="submit" variant="secondary" className="gap-2">
              <Search className="size-4" aria-hidden />
              <span className="hidden md:inline">Search</span>
            </Button>
          </form>
          <div className="flex items-center justify-end">
            <HealthIndicator />
          </div>
        </div>
        <nav className="flex items-center gap-4 text-sm font-medium md:hidden">
          <Link
            to="/"
            className={({ isActive }) =>
              cn(
                'transition-colors text-muted-foreground hover:text-foreground',
                isActive && 'text-foreground',
              )
            }
          >
            Recent
          </Link>
          <Link
            to="/search"
            className={({ isActive }) =>
              cn(
                'transition-colors text-muted-foreground hover:text-foreground',
                isActive && 'text-foreground',
              )
            }
          >
            Search
          </Link>
        </nav>
      </div>
    </header>
  )
}
