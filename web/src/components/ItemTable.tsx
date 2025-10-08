import { useQuery } from '@tanstack/react-query'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type Column,
  type ColumnDef,
  type ColumnFiltersState,
  type PaginationState,
  type SortingState,
} from '@tanstack/react-table'
import { ArrowDown, ArrowUp, ArrowUpDown, ExternalLink } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { Item } from '@/lib/api'
import { listRecentItems } from '@/lib/api'
import { queryKeys } from '@/lib/query'

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
  feedId?: string
  pageSize?: number
}

export default function ItemTable({ feedId, pageSize = DEFAULT_PAGE_SIZE }: ItemTableProps) {
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'published_at', desc: true },
  ])
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize,
  })
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(() =>
    feedId ? [{ id: 'feed_id', value: [feedId] }] : [],
  )

  useEffect(() => {
    setPagination((prev) => {
      if (prev.pageSize === pageSize) {
        return prev
      }
      return { pageIndex: 0, pageSize }
    })
  }, [pageSize])

  useEffect(() => {
    setColumnFilters((current) => {
      const existing = current.find((filter) => filter.id === 'feed_id')
      const nextValue = feedId ? [feedId] : []

      if (feedId) {
        if (
          existing &&
          Array.isArray(existing.value) &&
          arraysEqual(existing.value as string[], nextValue)
        ) {
          return current
        }
        return [
          ...current.filter((filter) => filter.id !== 'feed_id'),
          { id: 'feed_id', value: nextValue },
        ]
      }

      if (!existing) {
        return current
      }

      return current.filter((filter) => filter.id !== 'feed_id')
    })
  }, [feedId])

  useEffect(() => {
    setPagination((prev) => {
      if (prev.pageIndex === 0) {
        return prev
      }
      return { ...prev, pageIndex: 0 }
    })
  }, [sorting, columnFilters])

  const feedFilter = columnFilters.find((filter) => filter.id === 'feed_id')
  const feedIds = normalizeFilterValue(feedFilter?.value)
  const sortedFeedIds = [...feedIds].sort()
  const effectiveFeedIds = sortedFeedIds.length > 0 ? sortedFeedIds : undefined
  const sortState = sorting[0]
  const sortParam = sortState
    ? `${sortState.id}:${sortState.desc ? 'desc' : 'asc'}`
    : undefined
  const limit = pagination.pageSize
  const offset = pagination.pageIndex * pagination.pageSize

  const itemsQuery = useQuery({
    queryKey: queryKeys.recentItems({
      limit,
      offset,
      sort: sortParam,
      feed_id: effectiveFeedIds,
    }),
    queryFn: () =>
      listRecentItems({
        limit,
        offset,
        sort: sortParam,
        feed_id: effectiveFeedIds,
      }),
    keepPreviousData: false,
  })

  const items = itemsQuery.data?.items ?? []
  const total = itemsQuery.data?.total ?? 0
  const pageCount = limit > 0 ? Math.ceil(total / limit) : 0
  const rowCount = total
  const showSkeleton = itemsQuery.isLoading || itemsQuery.isFetching

  const table = useReactTable({
    data: items,
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: { sorting, pagination, columnFilters },
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    onColumnFiltersChange: setColumnFilters,
    manualSorting: true,
    manualPagination: true,
    manualFiltering: true,
    pageCount,
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
                items.length > 0 ? `${offset + 1}-${offset + items.length}` : 0
              } of ${rowCount} items`}
        </p>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage() || showSkeleton}
          >
            Previous
          </Button>
          <span className="text-xs text-muted-foreground">
            Page {pagination.pageIndex + 1} of {Math.max(pageCount, 1)}
          </span>
          <Button
            variant="outline"
            onClick={() => table.nextPage()}
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

function normalizeFilterValue(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is string => typeof item === 'string' && item.length > 0)
  }
  if (typeof value === 'string' && value.length > 0) {
    return [value]
  }
  return []
}

function arraysEqual(a: string[], b: string[]) {
  if (a.length !== b.length) {
    return false
  }
  return a.every((value, index) => value === b[index])
}
