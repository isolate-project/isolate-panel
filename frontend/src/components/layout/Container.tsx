import { ComponentChildren } from 'preact'
import { clsx } from 'clsx'

interface ContainerProps {
  children: ComponentChildren
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | 'full'
  className?: string
}

export function Container({
  children,
  maxWidth = 'full',
  className,
}: ContainerProps) {
  const maxWidthStyles = {
    sm: 'max-w-screen-sm',
    md: 'max-w-screen-md',
    lg: 'max-w-screen-lg',
    xl: 'max-w-screen-xl',
    '2xl': 'max-w-screen-2xl',
    full: 'max-w-full',
  }

  return (
    <div className={clsx('mx-auto px-4', maxWidthStyles[maxWidth], className)}>
      {children}
    </div>
  )
}
