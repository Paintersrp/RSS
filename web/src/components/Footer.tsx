import { Link } from '@tanstack/react-router'

import HealthIndicator from './HealthIndicator'
import { cn } from '@/lib/utils'

export default function Footer() {
  const year = new Date().getFullYear()
  return (
    <footer className="border-t border-border bg-muted/20">
      <div className="mx-auto flex max-w-6xl flex-col gap-4 px-4 py-6 text-sm text-muted-foreground md:flex-row md:items-center md:justify-between">
        <p>
          © {year} Courier ·{' '}
          <a
            href="https://github.com/orco-run/courier"
            target="_blank"
            rel="noreferrer"
            className="font-medium text-foreground transition-colors hover:text-primary"
          >
            View source
          </a>
        </p>
        <div className="flex flex-col gap-4 md:flex-row md:items-center">
          <nav className="flex items-center gap-4 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            <Link
              to="/"
              className={({ isActive }) =>
                cn(
                  'transition-colors hover:text-foreground',
                  isActive && 'text-foreground',
                )
              }
            >
              Recent
            </Link>
            <Link
              to="/search"
              className={({ isActive }) =>
                cn(
                  'transition-colors hover:text-foreground',
                  isActive && 'text-foreground',
                )
              }
            >
              Search
            </Link>
          </nav>
          <HealthIndicator className="justify-end" />
        </div>
      </div>
    </footer>
  )
}
