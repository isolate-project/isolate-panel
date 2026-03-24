import { clsx } from 'clsx'

interface SelectProps {
  name?: string
  value?: string | number
  options: Array<{ value: string | number; label: string }>
  placeholder?: string
  disabled?: boolean
  required?: boolean
  error?: string
  label?: string
  fullWidth?: boolean
  onChange?: (e: Event) => void
  className?: string
}

export function Select({
  name,
  value,
  options,
  placeholder,
  disabled = false,
  required = false,
  error,
  label,
  fullWidth = false,
  onChange,
  className,
}: SelectProps) {
  const selectStyles = clsx(
    'px-3 py-2 border rounded-lg transition-base',
    'bg-primary text-primary',
    'focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
    'disabled:opacity-60 disabled:cursor-not-allowed',
    error ? 'border-danger focus:ring-danger' : 'border-primary',
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
      <select
        name={name}
        value={value}
        disabled={disabled}
        required={required}
        onChange={onChange}
        className={selectStyles}
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
      {error && <p className="mt-1 text-sm text-danger">{error}</p>}
    </div>
  )
}
