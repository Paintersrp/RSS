import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type Column,
  type ColumnDef,
  type PaginationState,
  type SortingState,
  type Updater,
} from '@tanstack/react-table'
import { ArrowDown, ArrowUp, ArrowUpDown, ExternalLink } from 'lucide-react'
import { useMemo } from 'react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { Item } from '@/lib/api'
import {
  useRecentItemsQuery,
  type RecentItemsQueryState,
  type RecentItemsSort,
} from '@/lib/useRecentItemsQuery'

function formatDate(value?: string | null) {
  if (!value) {
    return 'Unknown'
  }
  try {
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(new Date(value))
  } catch (error) {
    return value
  }
}

const columns: ColumnDef<Item>[] = [
  {
    id: 'feed_id',
    accessorFn: (row) => row.feed_id,
    header: 'Feed',
    cell: ({ row }) => (
      <span className="font-medium text-sm text-muted-foreground">{
        row.original.feed_title
      }</span>
    ),
    enableSorting: false,
    enableColumnFilter: true,
  },
  {
    header: 'Title',
    accessorKey: 'title',
    cell: ({ row }) => (
      <a
        href={row.original.url}
        target="_blank"
        rel="noreferrer"
        className="flex items-center gap-2 text-sm font-semibold text-foreground transition-colors hover:text-primary"
      >
        {row.original.title}
        <ExternalLink className="size-3" aria-hidden />
      </a>
    ),
  },
  {
    header: 'Author',
    accessorKey: 'author',
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">
        {row.original.author ?? '—'}
      </span>
    ),
  },
  {
    header: ({ column }) => <SortableHeader column={column} label="Published" />,
    accessorKey: 'published_at',
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">
        {formatDate(row.original.published_at ?? row.original.retrieved_at)}
      </span>
    ),
    enableSorting: true,
  },
]

const DEFAULT_PAGE_SIZE = 20

interface ItemTableProps {
  feeds: string[]
  sort: RecentItemsSort
  page: number
  pageSize?: number
  onPageChange: (page: number) => void
  onSortChange: (sort: RecentItemsSort) => void
}

export default function ItemTable({
  feeds,
  sort,
  page,
  pageSize = DEFAULT_PAGE_SIZE,
  onPageChange,
  onSortChange,
}: ItemTableProps) {
  const queryState: RecentItemsQueryState = {
    feeds,
    sort,
    page,
    limit: pageSize,
  }

  const itemsQuery = useRecentItemsQuery(queryState, { keepPreviousData: true })

  const items = itemsQuery.data?.items ?? []
  const total = itemsQuery.data?.total ?? 0
  const pageCount = pageSize > 0 ? Math.ceil(total / pageSize) : 0
  const rowCount = total
  const showSkeleton = itemsQuery.isLoading || itemsQuery.isFetching

  const sortingState = useMemo<SortingState>(() => convertSortToState(sort), [sort])
  const paginationState = useMemo<PaginationState>(
    () => ({ pageIndex: Math.max(page - 1, 0), pageSize }),
    [page, pageSize],
  )
  const columnFilters = useMemo(() => {
    return feeds.length > 0
      ? [{ id: 'feed_id', value: [...feeds].filter(Boolean) }]
      : []
  }, [feeds])

  const table = useReactTable({
    data: items,
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: { sorting: sortingState, pagination: paginationState, columnFilters },
    manualSorting: true,
    manualPagination: true,
    manualFiltering: true,
    pageCount,
    onSortingChange: (updater) => handleSortingChange(updater, sort, onSortChange),
    onPaginationChange: (updater) =>
      handlePaginationChange(updater, paginationState, page, onPageChange),
  })

  return (
    <div className="space-y-4">
      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <table className="min-w-full divide-y divide-border">
          <thead className="bg-muted/60">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    scope="col"
                    className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-muted-foreground"
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-border">
            {showSkeleton ? (
              Array.from({ length: pagination.pageSize }).map((_, index) => (
                <tr key={`skeleton-${index}`} className="hover:bg-transparent">
                  <td colSpan={columns.length} className="px-4 py-3">
                    <Skeleton className="h-6 w-full" />
                  </td>
                </tr>
              ))
            ) : table.getRowModel().rows.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length}
                  className="px-4 py-6 text-center text-sm text-muted-foreground"
                >
                  No items yet. Add feeds to start populating the timeline.
                </td>
              </tr>
            ) : (
              table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="hover:bg-muted/40">
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="px-4 py-3 align-top">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">
          {showSkeleton
            ? 'Loading items…'
            : `Showing ${
                items.length > 0
                  ? `${paginationState.pageIndex * paginationState.pageSize + 1}-${
                      paginationState.pageIndex * paginationState.pageSize + items.length
                    }`
                  : 0
              } of ${rowCount} items`}
        </p>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() =>
              table.previousPage()
            }
            disabled={!table.getCanPreviousPage() || showSkeleton}
          >
            Previous
          </Button>
          <span className="text-xs text-muted-foreground">
            Page {paginationState.pageIndex + 1} of {Math.max(pageCount, 1)}
          </span>
          <Button
            variant="outline"
            onClick={() =>
              table.nextPage()
            }
            disabled={!table.getCanNextPage() || showSkeleton}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}

interface SortableHeaderProps {
  column: Column<Item, unknown>
  label: string
}

function SortableHeader({ column, label }: SortableHeaderProps) {
  const sorted = column.getIsSorted()
  const Icon = useMemo(() => {
    if (sorted === 'asc') return ArrowUp
    if (sorted === 'desc') return ArrowDown
    return ArrowUpDown
  }, [sorted])

  return (
    <button
      type="button"
      onClick={column.getToggleSortingHandler()}
      className="inline-flex items-center gap-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground"
    >
      {label}
      <Icon className="size-3" aria-hidden />
      <span className="sr-only">
        {sorted === 'asc'
          ? 'ascending'
          : sorted === 'desc'
            ? 'descending'
            : 'no sort'}
      </span>
    </button>
  )
}

function convertSortToState(sort: RecentItemsSort): SortingState {
  const [columnId, direction] = sort.split(':') as [string, string]
  return [
    {
      id: columnId,
      desc: direction !== 'asc',
    },
  ]
}

function handleSortingChange(
  updater: Updater<SortingState>,
  currentSort: RecentItemsSort,
  onSortChange: (next: RecentItemsSort) => void,
) {
  const nextState = typeof updater === 'function' ? updater(convertSortToState(currentSort)) : updater
  const next = nextState[0]
  if (!next) {
    if (currentSort !== 'published_at:desc') {
      onSortChange('published_at:desc')
    }
    return
  }
  const nextSort: RecentItemsSort = `published_at:${next.desc ? 'desc' : 'asc'}`
  if (nextSort !== currentSort) {
    onSortChange(nextSort)
  }
}

function handlePaginationChange(
  updater: Updater<PaginationState>,
  current: PaginationState,
  currentPage: number,
  onPageChange: (nextPage: number) => void,
) {
  const next = typeof updater === 'function' ? updater(current) : updater
  const nextPage = next.pageIndex + 1
  if (nextPage !== currentPage) {
    onPageChange(nextPage)
  }
}
