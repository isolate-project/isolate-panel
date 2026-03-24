import { clsx } from 'clsx'

interface CheckboxProps {
  name?: string
  checked?: boolean
  disabled?: boolean
  label?: string
  onChange?: (e: Event) => void
  className?: string
}

export function Checkbox({
  name,
  checked = false,
  disabled = false,
  label,
  onChange,
  className,
}: CheckboxProps) {
  return (
    <label
      className={clsx(
        'inline-flex items-center gap-2 cursor-pointer',
        disabled && 'opacity-60 cursor-not-allowed',
        className
      )}
    >
      <input
        type="checkbox"
        name={name}
        checked={checked}
        disabled={disabled}
        onChange={onChange}
        className={clsx(
          'w-4 h-4 rounded border-primary text-primary',
          'focus:ring-2 focus:ring-primary focus:ring-offset-0',
          'transition-base cursor-pointer',
          'disabled:cursor-not-allowed'
        )}
      />
      {label && <span className="text-sm text-primary">{label}</span>}
    </label>
  )
}
