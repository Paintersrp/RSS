import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from '@tanstack/react-table'
import { ExternalLink } from 'lucide-react'

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
    header: 'Feed',
    accessorKey: 'feed_title',
    cell: ({ row }) => (
      <span className="font-medium text-sm text-muted-foreground">{
        row.original.feed_title
      }</span>
    ),
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
    header: 'Published',
    accessorKey: 'published_at',
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">
        {formatDate(row.original.published_at ?? row.original.retrieved_at)}
      </span>
    ),
  },
]

interface ItemTableProps {
  items: Item[]
}

export default function ItemTable({ items }: ItemTableProps) {
  const table = useReactTable({
    data: items,
    columns,
    getCoreRowModel: getCoreRowModel(),
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
