import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { toast } from 'sonner'

import ItemCard from '@/components/ItemCard'
import ItemTable from '@/components/ItemTable'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { listRecentItems } from '@/lib/api'
import { queryKeys } from '@/lib/query'

const LIMIT = 50

export const Route = createFileRoute('/')({
  component: RecentItemsRoute,
})

function RecentItemsRoute() {
  const query = useQuery({
    queryKey: queryKeys.recentItems({ limit: LIMIT }),
    queryFn: () => listRecentItems({ limit: LIMIT }),
  })

  useEffect(() => {
    if (query.isError) {
      toast.error('Failed to load recent items. Try again?')
    }
  }, [query.isError])

  const items = query.data ?? []

  return (
    <div className="mx-auto w-full max-w-6xl space-y-10 px-4 py-10">
      <section className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Recent items</h1>
            <p className="text-sm text-muted-foreground">
              Showing the latest {items.length} stories fetched from your subscribed feeds.
            </p>
          </div>
          <Button onClick={() => query.refetch()} disabled={query.isFetching} variant="secondary">
            {query.isFetching ? 'Refreshing…' : 'Refresh'}
          </Button>
        </div>
        {query.isLoading ? (
          <RecentItemsSkeleton />
        ) : (
          <>
            <div className="hidden md:block">
              <ItemTable items={items} />
            </div>
            <div className="grid gap-6 md:hidden">
              {items.map((item) => (
                <ItemCard key={item.id} item={item} />
              ))}
            </div>
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

function RecentItemsSkeleton() {
  return (
    <div className="space-y-4">
      <div className="hidden flex-col gap-3 md:flex">
        {Array.from({ length: 5 }).map((_, index) => (
          <Skeleton key={index} className="h-12 w-full" />
        ))}
      </div>
      <div className="grid gap-6 md:hidden">
        {Array.from({ length: 4 }).map((_, index) => (
          <div key={index} className="space-y-3 rounded-xl border border-border bg-card p-6">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-6 w-3/4" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-9 w-24" />
          </div>
        ))}
      </div>
    </div>
  )
}
