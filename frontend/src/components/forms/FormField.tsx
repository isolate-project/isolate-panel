import { Input } from '../ui/Input'
import { Select } from '../ui/Select'
import { Checkbox } from '../ui/Checkbox'
import { Switch } from '../ui/Switch'

interface FormFieldProps {
  name: string
  label?: string
  type?: 'text' | 'email' | 'password' | 'number' | 'select' | 'checkbox' | 'switch'
  value?: string | number | boolean
  error?: string
  touched?: boolean
  required?: boolean
  disabled?: boolean
  placeholder?: string
  helperText?: string
  options?: Array<{ value: string | number; label: string }>
  onChange: (name: string, value: string | number | boolean) => void
  onBlur?: (name: string) => void
}

export function FormField({
  name,
  label,
  type = 'text',
  value,
  error,
  touched,
  required,
  disabled,
  placeholder,
  helperText,
  options,
  onChange,
  onBlur,
}: FormFieldProps) {
  const showError = touched && error

  const handleChange = (e: Event) => {
    const target = e.target as HTMLInputElement | HTMLSelectElement
    let newValue: string | number | boolean = target.value
    if (type === 'number') {
      newValue = target.value === '' ? '' : Number(target.value)
    }
    onChange(name, newValue)
  }

  const handleCheckboxChange = (e: Event) => {
    const target = e.target as HTMLInputElement
    onChange(name, target.checked)
  }

  const handleBlur = () => {
    onBlur?.(name)
  }

  if (type === 'select' && options) {
    return (
      <Select
        name={name}
        label={label}
        value={value as string}
        options={options}
        error={showError ? error : undefined}
        required={required}
        disabled={disabled}
        placeholder={placeholder}
        onChange={handleChange}
        fullWidth
      />
    )
  }

  if (type === 'checkbox') {
    return (
      <Checkbox
        name={name}
        label={label}
        checked={(value as boolean) || false}
        disabled={disabled}
        onChange={handleCheckboxChange}
      />
    )
  }

  if (type === 'switch') {
    return (
      <Switch
        name={name}
        label={label}
        checked={(value as boolean) || false}
        disabled={disabled}
        onChange={handleCheckboxChange}
      />
    )
  }

  return (
    <Input
      type={type as 'text' | 'email' | 'password' | 'number' | 'tel' | 'url'}
      name={name}
      label={label}
      value={(value as string | number) || ''}
      error={showError ? error : undefined}
      required={required}
      disabled={disabled}
      placeholder={placeholder}
      helperText={helperText}
      onChange={handleChange}
      onBlur={handleBlur}
      fullWidth
    />
  )
}
