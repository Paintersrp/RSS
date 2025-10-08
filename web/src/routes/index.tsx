import { createFileRoute } from '@tanstack/react-router'
import { useIsFetching, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo } from 'react'

import ItemCard from '@/components/ItemCard'
import ItemTable from '@/components/ItemTable'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import {
  ToggleGroup,
  ToggleGroupItem,
} from '@/components/ui/toggle-group'
import { listFeeds, type Feed } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import {
  buildQueryKey,
  useRecentItemsQuery,
  type RecentItemsSort,
} from '@/lib/useRecentItemsQuery'

const PAGE_SIZE = 20
const DEFAULT_SORT: RecentItemsSort = 'published_at:desc'

export const Route = createFileRoute('/')({
  validateSearch: (search) => ({
    feeds: parseFeeds(search.feeds),
    page: parsePage(search.page),
    sort: parseSort(search.sort),
    view: search.view === 'card' ? 'card' : 'table',
  }),
  component: RecentItemsRoute,
})

function RecentItemsRoute() {
  const { feeds, page, sort, view } = Route.useSearch()
  const navigate = Route.useNavigate()
  const queryClient = useQueryClient()

  const feedsQuery = useQuery({
    queryKey: queryKeys.feeds(),
    queryFn: () => listFeeds(),
    staleTime: 5 * 60 * 1000,
  })

  const allFeeds = feedsQuery.data ?? []
  const selectedFeedDetails = useMemo(
    () =>
      feeds
        .map((id) => allFeeds.find((feed) => feed.id === id))
        .filter((feed): feed is Feed => Boolean(feed)),
    [feeds, allFeeds],
  )

  const showTable = view !== 'card'
  const queryState = { feeds, page, sort, limit: PAGE_SIZE }
  const queryKey = buildQueryKey(queryState)
  const isItemsFetching = useIsFetching({ queryKey }) > 0

  const cardItemsQuery = useRecentItemsQuery(queryState, {
    keepPreviousData: true,
    enabled: !showTable,
  })
  const cardItems = cardItemsQuery.data?.items ?? []

  const feedSummary = getFeedSummary(selectedFeedDetails, feeds)
  const headline = showTable
    ? `Browsing page ${page} of recent stories ${feedSummary}`
    : cardItemsQuery.isLoading
      ? 'Loading the latest stories…'
      : `Showing the latest ${cardItems.length} stories ${feedSummary}`

  return (
    <div className="mx-auto w-full max-w-6xl space-y-10 px-4 py-10">
      <section className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Recent items</h1>
            <p className="text-sm text-muted-foreground">{headline}.</p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <FeedFilter
              feeds={allFeeds}
              selectedIds={feeds}
              disabled={feedsQuery.isLoading}
              onToggle={(feedId, checked) => {
                void navigate({
                  search: (prev) => {
                    const current = prev.feeds ?? []
                    const next = checked
                      ? Array.from(new Set([...current, feedId]))
                      : current.filter((value) => value !== feedId)
                    const normalized = next.length > 0 ? next.sort() : []
                    return {
                      ...prev,
                      feeds: normalized,
                      page: 1,
                    }
                  },
                })
              }}
              onClear={() => {
                void navigate({
                  search: (prev) => ({
                    ...prev,
                    feeds: [],
                    page: 1,
                  }),
                })
              }}
            />
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
              onClick={() => {
                void queryClient.invalidateQueries({ queryKey })
              }}
              disabled={isItemsFetching}
              variant="secondary"
            >
              {isItemsFetching ? 'Refreshing…' : 'Refresh'}
            </Button>
          </div>
        </div>
        {showTable ? (
          <ItemTable
            feeds={feeds}
            sort={sort}
            page={page}
            pageSize={PAGE_SIZE}
            onPageChange={(nextPage) => {
              const safePage = Number.isFinite(nextPage) && nextPage > 0 ? nextPage : 1
              void navigate({
                search: (prev) => ({
                  ...prev,
                  page: safePage,
                }),
              })
            }}
            onSortChange={(nextSort) => {
              void navigate({
                search: (prev) => ({
                  ...prev,
                  sort: nextSort,
                  page: 1,
                }),
              })
            }}
          />
        ) : cardItemsQuery.isLoading ? (
          <RecentItemsSkeleton view={view} />
        ) : (
          <>
            <div className="grid gap-6 md:grid-cols-2">
              {cardItems.map((item) => (
                <ItemCard key={item.id} item={item} />
              ))}
            </div>
            {cardItems.length === 0 && (
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

interface FeedFilterProps {
  feeds: Feed[]
  selectedIds: string[]
  disabled?: boolean
  onToggle: (feedId: string, checked: boolean) => void
  onClear: () => void
}

function FeedFilter({ feeds, selectedIds, disabled, onToggle, onClear }: FeedFilterProps) {
  const selectedFeedTitles = useMemo(
    () =>
      selectedIds
        .map((id) => feeds.find((feed) => feed.id === id))
        .filter((feed): feed is Feed => Boolean(feed))
        .map((feed) => feed.title || feed.url),
    [feeds, selectedIds],
  )

  const selectedCount = selectedIds.length
  const buttonLabel =
    selectedCount === 0
      ? 'All feeds'
      : selectedCount === 1
        ? selectedFeedTitles[0]
        : `${selectedCount} feeds selected`

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className="w-[220px] justify-between"
          disabled={disabled}
        >
          <span className="truncate text-left text-sm">{buttonLabel}</span>
          <span className="text-xs text-muted-foreground">
            {selectedCount === 0 ? '•' : `${selectedCount}`}
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-0" align="end">
        <div className="flex items-center justify-between border-b border-border px-3 py-2">
          <span className="text-sm font-medium text-foreground">Feeds</span>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onClear()}
            disabled={selectedCount === 0}
          >
            Clear
          </Button>
        </div>
        <div className="max-h-64 space-y-2 overflow-y-auto p-2">
          {feeds.length === 0 ? (
            <p className="px-2 py-6 text-center text-sm text-muted-foreground">
              No feeds available yet.
            </p>
          ) : (
            feeds.map((feed) => {
              const checked = selectedIds.includes(feed.id)
              const title = feed.title || feed.url
              return (
                <label
                  key={feed.id}
                  className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1.5 hover:bg-muted/60"
                >
                  <Checkbox
                    checked={checked}
                    onCheckedChange={(state) => onToggle(feed.id, state === true)}
                  />
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm font-medium text-foreground">{title}</span>
                    <span className="truncate text-xs text-muted-foreground">{feed.url}</span>
                  </span>
                </label>
              )
            })
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}

function parseFeeds(value: unknown): string[] {
  if (Array.isArray(value)) {
    return Array.from(
      new Set(
        value.filter((entry): entry is string => typeof entry === 'string' && entry.length > 0),
      ),
    ).sort()
  }
  if (typeof value === 'string' && value.length > 0) {
    return [value]
  }
  return []
}

function parsePage(value: unknown): number {
  const numeric = typeof value === 'string' ? Number.parseInt(value, 10) : Number(value)
  return Number.isFinite(numeric) && numeric > 0 ? numeric : 1
}

function parseSort(value: unknown): RecentItemsSort {
  if (value === 'published_at:asc' || value === 'published_at:desc') {
    return value
  }
  return DEFAULT_SORT
}

function getFeedSummary(selectedFeeds: Feed[], selectedIds: string[]) {
  if (selectedFeeds.length === 0) {
    return 'from your subscribed feeds'
  }
  if (selectedFeeds.length === 1) {
    return `from ${selectedFeeds[0].title || selectedFeeds[0].url}`
  }
  return `from ${selectedIds.length} selected feeds`
}
