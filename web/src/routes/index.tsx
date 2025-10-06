import * as React from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'

import { listItems, type Item } from '../lib/api'
import { Button } from '../components/ui/button'
import { ItemCard } from '../components/ItemCard'
import { ItemTable } from '../components/ItemTable'
import { Skeleton } from '../components/ui/skeleton'

export const Route = createFileRoute('/')({
  component: IndexRoute,
})

function IndexRoute() {
  const [view, setView] = React.useState<'table' | 'card'>('table')
  const { data, isLoading } = useQuery({ queryKey: ['items', view], queryFn: () => listItems({ limit: 50 }) })

  const items = data ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Recent items</h1>
        <div className="flex gap-2">
          <Button variant={view === 'table' ? 'default' : 'outline'} onClick={() => setView('table')}>
            Table
          </Button>
          <Button variant={view === 'card' ? 'default' : 'outline'} onClick={() => setView('card')}>
            Cards
          </Button>
        </div>
      </div>
      {isLoading ? (
        <LoadingState />
      ) : view === 'table' ? (
        <ItemTable items={items} />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {items.map((item: Item) => (
            <ItemCard key={item.id} item={item} />
          ))}
        </div>
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
