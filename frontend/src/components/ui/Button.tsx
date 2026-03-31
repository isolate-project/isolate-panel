import { ComponentProps } from 'preact'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'
import { Loader2 } from 'lucide-preact'

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-bg-primary transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 active:scale-[0.98]',
  {
    variants: {
      variant: {
        default: 'bg-primary text-white hover:bg-primary-hover shadow-sm',
        primary: 'bg-primary text-white hover:bg-primary-hover shadow-sm',
        destructive: 'bg-danger text-white hover:bg-red-600 shadow-sm',
        danger: 'bg-danger text-white hover:bg-red-600 shadow-sm',
        outline: 'border border-border-primary bg-transparent hover:bg-hover text-text-primary',
        secondary: 'bg-bg-tertiary text-text-primary hover:bg-bg-hover shadow-sm',
        ghost: 'hover:bg-hover hover:text-text-primary text-text-secondary',
        link: 'text-primary underline-offset-4 hover:underline',
        glass: 'glass-panel text-text-primary hover:bg-white/10 dark:hover:bg-white/5',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-md px-3 text-xs',
        lg: 'h-11 rounded-md px-8',
        icon: 'h-10 w-10',
      },
      fullWidth: {
        true: 'w-full',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
)

export interface ButtonProps
  extends Omit<ComponentProps<'button'>, 'size'>,
    VariantProps<typeof buttonVariants> {
  loading?: boolean
  icon?: preact.ComponentChild
}

export function Button({
  className,
  variant,
  size,
  fullWidth,
  loading,
  icon,
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(buttonVariants({ variant, size, fullWidth, className }))}
      disabled={disabled || loading}
      {...props}
    >
      {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
      {!loading && icon && <span className="mr-2" data-testid="icon">{icon}</span>}
      {children}
    </button>
  )
}
