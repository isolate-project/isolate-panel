import { ComponentProps } from 'preact'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const badgeVariants = cva(
  'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2',
  {
    variants: {
      variant: {
        default:
          'border-transparent bg-primary text-white shadow hover:bg-primary/80',
        secondary:
          'border-transparent bg-bg-tertiary text-text-primary hover:bg-bg-hover',
        destructive:
          'border-transparent bg-danger text-white shadow hover:bg-danger/80',
        success:
          'border-transparent bg-success text-white shadow hover:bg-success/80',
        warning:
          'border-transparent bg-warning text-text-primary shadow hover:bg-warning/80',
        danger:
          'border-transparent bg-danger text-white shadow hover:bg-danger/80',
        info:
          'border-transparent bg-blue-500 text-white shadow hover:bg-blue-600',
        outline: 'text-text-primary border-border-secondary',
        glass: 'glass-panel text-text-primary border-white/10 shadow-sm',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

export interface BadgeProps
  extends ComponentProps<'div'>,
    VariantProps<typeof badgeVariants> {
  showDot?: boolean
}

function Badge({ className, variant, showDot, children, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props}>
      {showDot && (
        <span 
          className={cn(
            "mr-1.5 h-1.5 w-1.5 rounded-full",
            variant === 'success' ? "bg-white" : 
            variant === 'destructive' ? "bg-white" : 
            variant === 'warning' ? "bg-black/50" : 
            variant === 'secondary' ? "bg-text-tertiary" : "bg-current"
          )} 
        />
      )}
      {children}
    </div>
  )
}

export { Badge, badgeVariants }
