import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'
import type { DateRange } from 'react-day-picker'

import ItemCard from '@/components/ItemCard'
import { Button } from '@/components/ui/button'
import { DateRangePicker } from '@/components/ui/date-picker'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { listFeeds, searchItems } from '@/lib/api'
import { queryKeys } from '@/lib/query'

const PAGE_SIZE = 20
const SAVED_SEARCHES = [
  { label: 'Rust', query: 'rust' },
  { label: 'Go', query: 'golang' },
  { label: 'AI', query: 'artificial intelligence' },
  { label: 'Security', query: 'security' },
  { label: 'Databases', query: 'postgres' },
]

export const Route = createFileRoute('/search')({
  validateSearch: (search) => ({
    q: typeof search.q === 'string' ? search.q : '',
    page: typeof search.page === 'number' && search.page > 0 ? search.page : 1,
    feed: typeof search.feed === 'string' ? search.feed : '',
    startDate: typeof search.startDate === 'string' ? search.startDate : undefined,
    endDate: typeof search.endDate === 'string' ? search.endDate : undefined,
  }),
  component: SearchRoute,
})

function SearchRoute() {
  const { q, page, feed, startDate, endDate } = Route.useSearch()
  const navigate = Route.useNavigate()
  const [term, setTerm] = useState(q)
  const [debouncedTerm, setDebouncedTerm] = useState(q)
  const [debounceSignal, setDebounceSignal] = useState(0)
  const offset = useMemo(() => (Math.max(1, page) - 1) * PAGE_SIZE, [page])
  const isFirstDebounce = useRef(true)
  const safeParseDate = useCallback((value?: string) => {
    if (!value) {
      return undefined
    }
    const parsed = new Date(value)
    return Number.isNaN(parsed.getTime()) ? undefined : parsed
  }, [])
  const [selectedRange, setSelectedRange] = useState<DateRange | undefined>(() => {
    const from = safeParseDate(startDate)
    const to = safeParseDate(endDate)
    if (!from && !to) {
      return undefined
    }
    return { from, to }
  })
  const selectedFromISO = selectedRange?.from?.toISOString() ?? ''
  const selectedToISO = selectedRange?.to?.toISOString() ?? ''

  const commitTerm = useCallback((value: string) => {
    setDebouncedTerm(value)
    setDebounceSignal((signal) => signal + 1)
  }, [])

  useEffect(() => {
    setTerm(q)
  }, [q])

  useEffect(() => {
    const from = safeParseDate(startDate)
    const to = safeParseDate(endDate)
    const fromISO = from?.toISOString() ?? ''
    const toISO = to?.toISOString() ?? ''

    if (selectedFromISO !== fromISO || selectedToISO !== toISO) {
      if (!from && !to) {
        setSelectedRange(undefined)
      } else {
        setSelectedRange({ from, to })
      }
    }
  }, [endDate, safeParseDate, selectedFromISO, selectedToISO, startDate])

  useEffect(() => {
    if (term === debouncedTerm) {
      return
    }

    const handler = window.setTimeout(() => {
      commitTerm(term)
    }, 300)

    return () => {
      window.clearTimeout(handler)
    }
  }, [commitTerm, debouncedTerm, term])

  useEffect(() => {
    if (isFirstDebounce.current) {
      isFirstDebounce.current = false
      return
    }

    const next = debouncedTerm.trim()

    if (next.length === 0) {
      if (q !== '' || page !== 1 || feed !== '' || startDate || endDate) {
        void navigate({
          search: {
            q: '',
            page: 1,
            feed: '',
            startDate,
            endDate,
          },
        })
      }
      return
    }

    if (next === q && page === 1) {
      return
    }

    void navigate({
      search: { q: next, page: 1, feed, startDate, endDate },
    })
  }, [debouncedTerm, debounceSignal, endDate, feed, navigate, page, q, startDate])

  const feedsQuery = useQuery({
    queryKey: queryKeys.feeds(),
    queryFn: () => listFeeds(),
    staleTime: 5 * 60 * 1000,
    onError: (error) => {
      console.error('Failed to load feeds', error)
      const message =
        error instanceof Error ? error.message : 'Please try again in a few moments.'
      toast.error('Unable to load feeds', {
        id: 'feeds-error',
        description: `${message} Retry shortly or contact support if the issue persists.`,
      })
    },
    onSuccess: () => {
      toast.dismiss('feeds-error')
    },
  })

  const startDateISO = startDate ?? undefined
  const endDateISO = endDate ?? undefined

  const query = useQuery({
    queryKey: queryKeys.search({
      query: q,
      limit: PAGE_SIZE,
      offset,
      feed_id: feed || undefined,
      startDate: startDateISO,
      endDate: endDateISO,
    }),
    queryFn: () => searchItems({
      query: q,
      limit: PAGE_SIZE,
      offset,
      feed_id: feed || undefined,
      startDate: startDateISO,
      endDate: endDateISO,
    }),
    enabled: q.trim().length > 0,
    keepPreviousData: true,
    onError: (error) => {
      console.error('Search request failed', error)
      const message =
        error instanceof Error ? error.message : 'Please try again in a few moments.'
      toast.error('Unable to complete search', {
        id: 'search-error',
        description: `${message} Retry your search or adjust the filters before trying again.`,
      })
    },
    onSuccess: () => {
      toast.dismiss('search-error')
    },
  })

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const next = term.trim()
    setTerm(next)
    commitTerm(next)
  }

  const goToPage = (nextPage: number) => {
    void navigate({
      search: { q, page: nextPage, feed, startDate, endDate },
    })
  }

  const results = query.data?.hits ?? []
  const total = query.data?.estimated_total ?? 0
  const hasQuery = q.trim().length > 0
  const hasNext = offset + results.length < total
  const hasPrev = page > 1
  const feeds = feedsQuery.data ?? []
  const activeFeed = feed ? feeds.find((f) => f.id === feed) : undefined
  const feedSelection = feed || 'all'
  const hasDateRange = Boolean(selectedRange?.from || selectedRange?.to)

  const handleDateRangeChange = (range: DateRange | undefined) => {
    setSelectedRange(range)
    void navigate({
      search: {
        q,
        page: 1,
        feed,
        startDate: range?.from ? range.from.toISOString() : undefined,
        endDate: range?.to ? range.to.toISOString() : undefined,
      },
    })
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-10 px-4 py-10">
      <section className="space-y-6">
        <div className="space-y-3">
          <h1 className="text-2xl font-semibold text-foreground">Search</h1>
          <p className="text-sm text-muted-foreground">
            Query the Meilisearch index that powers Courier. Results update as new items are ingested.
          </p>
        </div>
        <form onSubmit={handleSubmit} className="flex flex-col gap-3 md:flex-row md:items-center" role="search">
          <Input
            value={term}
            onChange={(event) => setTerm(event.target.value)}
            placeholder="Search for keywords, authors, or feed titles…"
            aria-label="Search all items"
          />
          <div className="flex items-center gap-2">
            <Button type="submit" disabled={!term.trim()}>
              Search
            </Button>
            {hasQuery && (
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setTerm('')
                  commitTerm('')
                }}
              >
                Clear
              </Button>
            )}
          </div>
        </form>

        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <span>Saved searches:</span>
            <div className="flex flex-wrap gap-2">
              {SAVED_SEARCHES.map((item) => (
                <Button
                  key={item.query}
                  size="sm"
                  variant={q.toLowerCase() === item.query.toLowerCase() ? 'default' : 'outline'}
                  onClick={() => {
                    setTerm(item.query)
                    commitTerm(item.query)
                  }}
                >
                  {item.label}
                </Button>
              ))}
            </div>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-end">
            <div className="flex items-center gap-2">
              <DateRangePicker
                value={selectedRange}
                onChange={handleDateRangeChange}
                placeholder="Any time"
              />
              {hasDateRange && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDateRangeChange(undefined)}
                >
                  Clear dates
                </Button>
              )}
            </div>
            <Select
              value={feedSelection}
              onValueChange={(value) => {
                const nextFeed = value === 'all' ? '' : value
                void navigate({
                  search: { q, page: 1, feed: nextFeed, startDate, endDate },
                })
              }}
              disabled={feedsQuery.isLoading}
            >
              <SelectTrigger className="w-[220px]">
                <SelectValue placeholder="All feeds" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All feeds</SelectItem>
                {feeds.map((f) => (
                  <SelectItem key={f.id} value={f.id}>
                    {f.title || f.url}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        {!hasQuery ? (
          <EmptyState />
        ) : query.isLoading ? (
          <SearchSkeleton />
        ) : (
          <div className="space-y-6">
            <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
              <p className="text-sm text-muted-foreground">
                Showing {results.length} of {total} results for <span className="font-medium text-foreground">“{q}”</span>
                {activeFeed && (
                  <>
                    {' '}from <span className="font-medium text-foreground">{activeFeed.title}</span>
                  </>
                )}
              </p>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span>Page {page}</span>
                <span aria-hidden>·</span>
                <span>Index updated live from the API</span>
              </div>
            </div>
            {results.length === 0 ? (
              <p className="rounded-lg border border-dashed border-border bg-muted/30 p-6 text-center text-sm text-muted-foreground">
                No matches yet. Try broadening your terms or wait for the next crawl.
              </p>
            ) : (
              <div className="grid gap-6 md:grid-cols-2">
                {results.map((item) => (
                  <ItemCard
                    key={item.id}
                    item={{
                      id: item.id,
                      feed_id: item.feed_id,
                      feed_title: item.feed_title,
                      guid: null,
                      url: item.url,
                      title: item.title,
                      author: null,
                      content_html: item.content_text,
                      content_text: item.content_text,
                      published_at: item.published_at ?? null,
                      retrieved_at: item.published_at ?? new Date().toISOString(),
                    }}
                  />
                ))}
              </div>
            )}
            {(hasPrev || hasNext) && (
              <div className="flex items-center justify-between">
                <Button
                  variant="outline"
                  onClick={() => goToPage(page - 1)}
                  disabled={!hasPrev || query.isFetching}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  onClick={() => goToPage(page + 1)}
                  disabled={!hasNext || query.isFetching}
                >
                  Next
                </Button>
              </div>
            )}
          </div>
        )}
      </section>
    </div>
  )
}

function EmptyState() {
  return (
    <div className="rounded-lg border border-dashed border-border bg-muted/30 p-10 text-center text-muted-foreground">
      <p className="text-sm">
        Start typing above to search across everything Courier has ingested.
      </p>
    </div>
  )
}

function SearchSkeleton() {
  return (
    <div className="grid gap-6 md:grid-cols-2">
      {Array.from({ length: 6 }).map((_, index) => (
        <div key={index} className="space-y-3 rounded-xl border border-border bg-card p-6">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-6 w-3/4" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-5/6" />
          <Skeleton className="h-9 w-28" />
        </div>
      ))}
    </div>
  )
}
