import { ComponentChildren } from 'preact'
import { clsx } from 'clsx'

interface CardProps {
  children: ComponentChildren
  className?: string
  padding?: 'none' | 'sm' | 'md' | 'lg'
  hover?: boolean
}

export function Card({
  children,
  className,
  padding = 'md',
  hover = false,
}: CardProps) {
  const paddingStyles = {
    none: '',
    sm: 'p-3',
    md: 'p-4',
    lg: 'p-6',
  }

  return (
    <div
      className={clsx(
        'bg-primary border border-primary rounded-lg shadow-sm',
        paddingStyles[padding],
        hover && 'hover:shadow-md transition-base cursor-pointer',
        className
      )}
    >
      {children}
    </div>
  )
}
