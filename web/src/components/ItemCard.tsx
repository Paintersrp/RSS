import { type Item, type SearchHit } from '../lib/api'
import { Card, CardContent, CardHeader } from './ui/card'

type ItemLike = Item | SearchHit

export function ItemCard({ item }: { item: ItemLike }) {
  return (
    <Card key={item.id} className="space-y-3">
      <CardHeader>
        <a
          href={item.url ?? '#'}
          target="_blank"
          rel="noreferrer"
          className="text-lg font-semibold hover:underline"
        >
          {item.title}
        </a>
        <p className="text-sm text-muted-foreground">{item.feed_title ?? 'Unknown feed'}</p>
      </CardHeader>
      <CardContent>
        <p className="text-sm leading-relaxed text-muted-foreground">{item.content_text.slice(0, 280)}...</p>
        <p className="mt-3 text-xs text-muted-foreground">
          {item.published_at ? new Date(item.published_at).toLocaleString() : 'Unpublished'}
        </p>
      </CardContent>
    </Card>
  )
}
