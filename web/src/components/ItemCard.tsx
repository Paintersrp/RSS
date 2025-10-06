import { ExternalLink } from 'lucide-react'

import type { Item } from '@/lib/api'
import { Button } from './ui/button'
import { cn } from '@/lib/utils'

interface ItemCardProps {
  item: Item
  className?: string
}

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

function truncate(text: string, maxLength = 240) {
  if (text.length <= maxLength) {
    return text
  }
  return `${text.slice(0, maxLength).trimEnd()}…`
}

export default function ItemCard({ item, className }: ItemCardProps) {
  const description = truncate(item.content_text || item.content_html || '')
  const publishedAt = item.published_at ?? item.retrieved_at

  return (
    <article
      className={cn(
        'flex h-full flex-col justify-between gap-4 rounded-xl border border-border bg-card p-6 shadow-sm transition-shadow hover:shadow-lg',
        className,
      )}
    >
      <div className="space-y-3">
        <div className="flex flex-wrap items-center gap-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
          <span className="rounded-full bg-muted px-2 py-1 text-[11px] text-muted-foreground">
            {item.feed_title}
          </span>
          <span>{formatDate(publishedAt)}</span>
        </div>
        <h3 className="text-lg font-semibold leading-tight text-foreground">
          <a
            href={item.url}
            target="_blank"
            rel="noreferrer"
            className="transition-colors hover:text-primary"
          >
            {item.title}
          </a>
        </h3>
        {description && (
          <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        {item.author && <span>By {item.author}</span>}
        <Button asChild variant="outline" size="sm">
          <a href={item.url} target="_blank" rel="noreferrer">
            View original
            <ExternalLink className="ml-2 size-3.5" aria-hidden />
          </a>
        </Button>
      </div>
    </article>
  )
}
