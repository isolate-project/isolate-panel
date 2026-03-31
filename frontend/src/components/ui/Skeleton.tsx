import { ComponentProps } from 'preact'
import { cn } from '../../lib/utils'

interface SkeletonProps extends ComponentProps<'div'> {
  variant?: string
}

function Skeleton({
  className,
  variant,
  ...props
}: SkeletonProps) {
  return (
    <div
      className={cn('animate-pulse rounded-md bg-bg-tertiary', className)}
      {...props}
    />
  )
}

export { Skeleton }
