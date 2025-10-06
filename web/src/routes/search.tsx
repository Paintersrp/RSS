import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'

import { searchItems } from '../lib/api'
import { ItemCard } from '../components/ItemCard'
import { Skeleton } from '../components/ui/skeleton'

export const Route = createFileRoute('/search')({
  validateSearch: (search: Record<string, unknown>) => ({
    q: typeof search.q === 'string' ? search.q : '',
  }),
  component: SearchRoute,
})

function SearchRoute() {
  const { q } = Route.useSearch()
  const queryEnabled = q.length > 0
  const { data, isLoading } = useQuery({
    queryKey: ['search', q],
    queryFn: () => searchItems({ q, limit: 40 }),
    enabled: queryEnabled,
  })

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Search results for “{q}”</h1>
      {!queryEnabled ? (
        <p className="text-sm text-muted-foreground">Enter a term above to search the index.</p>
      ) : isLoading ? (
        <LoadingState />
      ) : data && data.hits.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2">
          {data.hits.map((item) => (
            <ItemCard key={item.id} item={item} />
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">No results yet.</p>
      )}
    </div>
  )
}

function LoadingState() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-32 w-full" />
    </div>
  )
}
