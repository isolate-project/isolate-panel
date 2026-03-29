import { ComponentChildren } from 'preact'
import { cn } from '../../lib/utils'
import { Input } from '../ui/Input'
import { Switch } from '../ui/Switch'
import { Select } from '../ui/Select'

interface FormFieldProps {
  name: string
  label: string
  type: 'text' | 'email' | 'password' | 'number' | 'switch' | 'select'
  value: string | number | boolean | undefined
  onChange: (name: string, value: string | number | boolean) => void
  onBlur?: (name: string) => void
  error?: string
  touched?: boolean
  required?: boolean
  disabled?: boolean
  placeholder?: string
  helperText?: string
  className?: string
  options?: { value: string; label: string }[]
  children?: ComponentChildren
}

export function FormField({
  name,
  label,
  type,
  value,
  onChange,
  onBlur,
  error,
  touched,
  required,
  disabled,
  placeholder,
  helperText,
  className,
  options,
  children,
}: FormFieldProps) {
  const hasError = touched && !!error

  const handleChange = (e: Event) => {
    const target = e.target as HTMLInputElement | HTMLSelectElement
    let val: string | number | boolean = target.value
    if (type === 'number') {
      val = Number(target.value)
    }
    onChange(name, val)
  }

  const handleBlur = () => {
    if (onBlur) onBlur(name)
  }

  if (type === 'switch') {
    return (
      <div className={cn("flex flex-row items-center justify-between rounded-xl border border-border-primary bg-bg-secondary/50 p-4 shadow-sm", className)}>
        <div className="space-y-1 pr-6">
          <label className="text-sm font-medium text-text-primary">
            {label}
            {required && <span className="text-color-danger ml-1">*</span>}
          </label>
          {helperText && <p className="text-xs text-text-secondary leading-snug">{helperText}</p>}
          {hasError && <p className="text-xs text-color-danger mt-1">{error}</p>}
        </div>
        <Switch
          checked={Boolean(value)}
          onChange={(checked) => onChange(name, checked)}
          disabled={disabled}
        />
      </div>
    )
  }

  return (
    <div className={cn("space-y-2", className)}>
      <label htmlFor={name} className="text-sm font-medium text-text-primary">
        {label}
        {required && <span className="text-color-danger ml-1">*</span>}
      </label>

      {children ? (
        children
      ) : type === 'select' ? (
        <Select
          id={name}
          name={name}
          value={String(value || '')}
          disabled={disabled}
          onChange={handleChange}
          onBlur={handleBlur}
          options={options || []}
          className={hasError ? "border-color-danger ring-color-danger/20" : ""}
        />
      ) : (
        <Input
          id={name}
          name={name}
          type={type}
          value={(value as string | number) || ''}
          placeholder={placeholder}
          disabled={disabled}
          isInvalid={hasError}
          onChange={handleChange}
          onBlur={handleBlur}
          className="bg-bg-primary text-text-primary"
        />
      )}

      {hasError ? (
        <p className="text-xs font-medium text-color-danger">{error}</p>
      ) : helperText ? (
        <p className="text-xs text-text-secondary">{helperText}</p>
      ) : null}
    </div>
  )
}
