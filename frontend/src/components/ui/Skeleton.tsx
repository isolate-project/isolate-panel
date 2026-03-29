import { ComponentProps } from 'preact'
import { cn } from '../../lib/utils'

function Skeleton({
  className,
  ...props
}: ComponentProps<'div'>) {
  return (
    <div
      className={cn('animate-pulse rounded-md bg-bg-tertiary', className)}
      {...props}
    />
  )
}

export { Skeleton }
