import { ComponentProps } from 'preact'
import { cn } from '../../lib/utils'
import { ChevronDown } from 'lucide-preact'

export interface SelectProps extends Omit<ComponentProps<'select'>, 'size'> {
  options: { value: string; label: string }[]
  placeholder?: string
  label?: string
  error?: string
  fullWidth?: boolean
}

export function Select({
  options,
  placeholder,
  label,
  error,
  fullWidth,
  className,
  ...props
}: SelectProps) {
  return (
    <div className={fullWidth ? 'w-full' : ''}>
      {label && (
        <label className="block text-sm font-medium text-text-primary mb-1">
          {label}
        </label>
      )}
      <div className="relative">
        <select
          className={cn(
            'w-full appearance-none rounded-lg border border-border-primary bg-bg-primary px-3 py-2 pr-10 text-text-primary',
            'focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
            'disabled:opacity-60 disabled:cursor-not-allowed',
            error && 'border-danger focus:ring-danger',
            className
          )}
          {...props}
        >
          {placeholder && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <ChevronDown className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-tertiary pointer-events-none" />
      </div>
      {error && <p className="mt-1 text-sm text-danger">{error}</p>}
    </div>
  )
}
