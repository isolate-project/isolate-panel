import { clsx } from 'clsx'

interface InputProps {
  type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url'
  id?: string
  name?: string
  value?: string | number
  placeholder?: string
  disabled?: boolean
  required?: boolean
  error?: string
  label?: string
  helperText?: string
  fullWidth?: boolean
  isInvalid?: boolean
  onChange?: (e: Event) => void
  onBlur?: (e: Event) => void
  className?: string
  min?: string | number
  max?: string | number
  autoFocus?: boolean
}

export function Input({
  type = 'text',
  name,
  value,
  placeholder,
  disabled = false,
  required = false,
  error,
  label,
  helperText,
  fullWidth = false,
  isInvalid,
  onChange,
  onBlur,
  className,
  min,
  max,
  autoFocus,
}: InputProps) {
  const hasError = error || isInvalid

  const inputStyles = clsx(
    'px-3 py-2 border rounded-lg transition-base',
    'bg-bg-primary text-text-primary',
    'placeholder:text-text-tertiary',
    'focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
    'disabled:opacity-60 disabled:cursor-not-allowed',
    hasError
      ? 'border-danger focus:ring-danger'
      : 'border-border-primary',
    fullWidth ? 'w-full' : '',
    className
  )

  const errorId = name ? `${name}-error` : undefined
  const helperId = name ? `${name}-helper` : undefined

  return (
    <div className={fullWidth ? 'w-full' : ''}>
      {label && (
        <label htmlFor={name} className="block text-sm font-medium text-text-primary mb-1">
          {label}
          {required && <span className="text-danger ml-1">*</span>}
        </label>
      )}
      <input
        type={type}
        id={name}
        name={name}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        required={required}
        onChange={onChange}
        onBlur={onBlur}
        min={min}
        max={max}
        autoFocus={autoFocus}
        className={inputStyles}
        aria-invalid={hasError ? 'true' : undefined}
        aria-describedby={
          hasError && errorId ? errorId : helperText && helperId ? helperId : undefined
        }
      />
      {hasError && (
        <p id={errorId} className="mt-1 text-sm text-danger" role="alert">{error}</p>
      )}
      {helperText && !hasError && (
        <p id={helperId} className="mt-1 text-sm text-text-secondary">{helperText}</p>
      )}
    </div>
  )
}
