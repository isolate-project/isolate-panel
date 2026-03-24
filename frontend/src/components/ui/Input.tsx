import { clsx } from 'clsx'

interface InputProps {
  type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url'
  name?: string
  value?: string | number
  placeholder?: string
  disabled?: boolean
  required?: boolean
  error?: string
  label?: string
  helperText?: string
  fullWidth?: boolean
  onChange?: (e: Event) => void
  onBlur?: (e: Event) => void
  className?: string
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
  onChange,
  onBlur,
  className,
}: InputProps) {
  const inputStyles = clsx(
    'px-3 py-2 border rounded-lg transition-base',
    'bg-primary text-primary',
    'placeholder:text-tertiary',
    'focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
    'disabled:opacity-60 disabled:cursor-not-allowed',
    error
      ? 'border-danger focus:ring-danger'
      : 'border-primary',
    fullWidth ? 'w-full' : '',
    className
  )

  return (
    <div className={fullWidth ? 'w-full' : ''}>
      {label && (
        <label className="block text-sm font-medium text-primary mb-1">
          {label}
          {required && <span className="text-danger ml-1">*</span>}
        </label>
      )}
      <input
        type={type}
        name={name}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        required={required}
        onChange={onChange}
        onBlur={onBlur}
        className={inputStyles}
      />
      {error && (
        <p className="mt-1 text-sm text-danger">{error}</p>
      )}
      {helperText && !error && (
        <p className="mt-1 text-sm text-secondary">{helperText}</p>
      )}
    </div>
  )
}
