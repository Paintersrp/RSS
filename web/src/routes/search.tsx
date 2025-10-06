import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { FormEvent, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

import ItemCard from '@/components/ItemCard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { searchItems } from '@/lib/api'
import { queryKeys } from '@/lib/query'

const PAGE_SIZE = 20

export const Route = createFileRoute('/search')({
  validateSearch: (search) => ({
    q: typeof search.q === 'string' ? search.q : '',
    page: typeof search.page === 'number' && search.page > 0 ? search.page : 1,
  }),
  component: SearchRoute,
})

function SearchRoute() {
  const { q, page } = Route.useSearch()
  const navigate = Route.useNavigate()
  const [term, setTerm] = useState(q)
  const offset = useMemo(() => (Math.max(1, page) - 1) * PAGE_SIZE, [page])

  useEffect(() => {
    setTerm(q)
  }, [q])

  const query = useQuery({
    queryKey: queryKeys.search({ query: q, limit: PAGE_SIZE, offset }),
    queryFn: () => searchItems({ query: q, limit: PAGE_SIZE, offset }),
    enabled: q.trim().length > 0,
    keepPreviousData: true,
  })

  useEffect(() => {
    if (query.isError) {
      toast.error('Unable to run search. Please try again.')
    }
  }, [query.isError])

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const next = term.trim()
    void navigate({
      search: { q: next, page: 1 },
    })
  }

  const goToPage = (nextPage: number) => {
    void navigate({
      search: { q, page: nextPage },
    })
  }

  const results = query.data?.hits ?? []
  const total = query.data?.estimated_total ?? 0
  const hasQuery = q.trim().length > 0
  const hasNext = offset + results.length < total
  const hasPrev = page > 1

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
                  void navigate({ search: { q: '', page: 1 } })
                }}
              >
                Clear
              </Button>
            )}
          </div>
        </form>

        {!hasQuery ? (
          <EmptyState />
        ) : query.isLoading ? (
          <SearchSkeleton />
        ) : (
          <div className="space-y-6">
            <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
              <p className="text-sm text-muted-foreground">
                Showing {results.length} of {total} results for <span className="font-medium text-foreground">“{q}”</span>
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
