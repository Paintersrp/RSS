import { Toaster as SonnerToaster, type ToasterProps } from 'sonner'

export function Toaster(props: ToasterProps) {
  return (
    <SonnerToaster
      toastOptions={{
        classNames: {
          toast: 'bg-popover text-popover-foreground border border-border shadow-lg',
          description: 'text-sm text-muted-foreground',
          actionButton:
            'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors',
          cancelButton:
            'bg-muted text-muted-foreground hover:bg-muted/80 transition-colors',
        },
      }}
      {...props}
    />
  )
}
