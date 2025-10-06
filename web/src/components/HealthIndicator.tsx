import { useQuery } from '@tanstack/react-query'

import { getHealth } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { cn } from '@/lib/utils'

interface HealthIndicatorProps {
  className?: string
}

export default function HealthIndicator({
  className,
}: HealthIndicatorProps) {
  const {
    data,
    isError,
    isFetching,
  } = useQuery({
    queryKey: queryKeys.health(),
    queryFn: getHealth,
    refetchInterval: 60_000,
    staleTime: 30_000,
  })

  let statusLabel = 'Checking…'
  let dotClass = 'bg-muted-foreground'

  if (isError) {
    statusLabel = 'Offline'
    dotClass = 'bg-destructive'
  } else if (data?.status === 'ok') {
    statusLabel = 'Online'
    dotClass = 'bg-emerald-500'
  } else if (data?.status) {
    statusLabel = data.status
    dotClass = 'bg-amber-500'
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-2 text-sm text-muted-foreground',
        className,
      )}
    >
      <span
        className={cn(
          'h-2.5 w-2.5 rounded-full',
          dotClass,
          isFetching && 'animate-pulse',
        )}
        aria-hidden
      />
      <span>{statusLabel}</span>
    </span>
  )
}
