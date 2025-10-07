import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type Column,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table'
import { ArrowDown, ArrowUp, ArrowUpDown, ExternalLink } from 'lucide-react'
import { useMemo, useState } from 'react'

import type { Item } from '@/lib/api'

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
    header: ({ column }) => <SortableHeader column={column} label="Feed" />,
    accessorKey: 'feed_title',
    cell: ({ row }) => (
      <span className="font-medium text-sm text-muted-foreground">{
        row.original.feed_title
      }</span>
    ),
    enableSorting: true,
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
    sortingFn: (rowA, rowB) => {
      const valueA = rowA.original.published_at ?? rowA.original.retrieved_at
      const valueB = rowB.original.published_at ?? rowB.original.retrieved_at
      if (!valueA && !valueB) return 0
      if (!valueA) return -1
      if (!valueB) return 1
      const timeA = Date.parse(valueA)
      const timeB = Date.parse(valueB)
      return timeA - timeB
    },
  },
]

interface ItemTableProps {
  items: Item[]
}

export default function ItemTable({ items }: ItemTableProps) {
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'published_at', desc: true },
  ])
  const table = useReactTable({
    data: items,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    state: { sorting },
    onSortingChange: setSorting,
    manualSorting: false,
  })

  return (
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
          {table.getRowModel().rows.length === 0 ? (
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
