import { ComponentChildren } from 'preact'

interface PageHeaderProps {
  title: string
  description?: string
  actions?: ComponentChildren
}

export function PageHeader({ title, description, actions }: PageHeaderProps) {
  return (
    <div className="mb-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-primary">{title}</h1>
          {description && (
            <p className="mt-1 text-sm text-secondary">{description}</p>
          )}
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </div>
    </div>
  )
}
