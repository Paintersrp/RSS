import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'

import ItemCard from '@/components/ItemCard'
import ItemTable from '@/components/ItemTable'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  ToggleGroup,
  ToggleGroupItem,
} from '@/components/ui/toggle-group'
import { listFeeds, listRecentItems } from '@/lib/api'
import { queryKeys } from '@/lib/query'

const LIMIT = 50

export const Route = createFileRoute('/')({
  validateSearch: (search) => ({
    feed: typeof search.feed === 'string' ? search.feed : '',
    view: search.view === 'card' ? 'card' : 'table',
  }),
  component: RecentItemsRoute,
})

function RecentItemsRoute() {
  const { feed, view } = Route.useSearch()
  const navigate = Route.useNavigate()

  const feedsQuery = useQuery({
    queryKey: queryKeys.feeds(),
    queryFn: () => listFeeds(),
    staleTime: 5 * 60 * 1000,
  })

  const itemsQuery = useQuery({
    queryKey: queryKeys.recentItems({ limit: LIMIT, feed_id: feed || undefined }),
    queryFn: () => listRecentItems({ limit: LIMIT, feed_id: feed || undefined }),
  })

  const items = itemsQuery.data ?? []
  const feeds = feedsQuery.data ?? []
  const activeFeed = feed ? feeds.find((f) => f.id === feed) : undefined
  const feedSelection = feed || 'all'
  const showTable = view !== 'card'

  return (
    <div className="mx-auto w-full max-w-6xl space-y-10 px-4 py-10">
      <section className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Recent items</h1>
            <p className="text-sm text-muted-foreground">
              Showing the latest {items.length} stories
              {activeFeed ? ` from ${activeFeed.title}` : ' from your subscribed feeds'}.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Select
              value={feedSelection}
              onValueChange={(value) => {
                const nextFeed = value === 'all' ? '' : value
                void navigate({
                  search: (prev) => ({ ...prev, feed: nextFeed || undefined }),
                })
              }}
              disabled={feedsQuery.isLoading}
            >
              <SelectTrigger className="w-[200px]">
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
            <ToggleGroup
              type="single"
              value={view}
              onValueChange={(next) => {
                if (!next) return
                void navigate({
                  search: (prev) => ({ ...prev, view: next as 'table' | 'card' }),
                })
              }}
            >
              <ToggleGroupItem value="table" aria-label="Table view">
                Table
              </ToggleGroupItem>
              <ToggleGroupItem value="card" aria-label="Card view">
                Cards
              </ToggleGroupItem>
            </ToggleGroup>
            <Button
              onClick={() => itemsQuery.refetch()}
              disabled={itemsQuery.isFetching}
              variant="secondary"
            >
              {itemsQuery.isFetching ? 'Refreshing…' : 'Refresh'}
            </Button>
          </div>
        </div>
        {itemsQuery.isLoading ? (
          <RecentItemsSkeleton view={view} />
        ) : (
          <>
            {showTable ? (
              <ItemTable items={items} />
            ) : (
              <div className="grid gap-6 md:grid-cols-2">
                {items.map((item) => (
                  <ItemCard key={item.id} item={item} />
                ))}
              </div>
            )}
            {items.length === 0 && (
              <p className="text-center text-sm text-muted-foreground">
                No items yet. Add feeds via the API to start filling the list.
              </p>
            )}
          </>
        )}
      </section>
    </div>
  )
}

interface RecentItemsSkeletonProps {
  view: 'table' | 'card'
}

function RecentItemsSkeleton({ view }: RecentItemsSkeletonProps) {
  if (view === 'card') {
    return (
      <div className="grid gap-6 md:grid-cols-2">
        {Array.from({ length: 6 }).map((_, index) => (
          <div key={index} className="space-y-3 rounded-xl border border-border bg-card p-6">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-6 w-3/4" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-9 w-24" />
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton key={index} className="h-12 w-full" />
      ))}
    </div>
  )
}
