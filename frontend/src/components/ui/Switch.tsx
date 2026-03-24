import { clsx } from 'clsx'

interface SwitchProps {
  name?: string
  checked?: boolean
  disabled?: boolean
  label?: string
  onChange?: (e: Event) => void
  className?: string
}

export function Switch({
  name,
  checked = false,
  disabled = false,
  label,
  onChange,
  className,
}: SwitchProps) {
  return (
    <label
      className={clsx(
        'inline-flex items-center gap-3 cursor-pointer',
        disabled && 'opacity-60 cursor-not-allowed',
        className
      )}
    >
      <div className="relative">
        <input
          type="checkbox"
          name={name}
          checked={checked}
          disabled={disabled}
          onChange={onChange}
          className="sr-only peer"
        />
        <div
          className={clsx(
            'w-11 h-6 rounded-full transition-base',
            'peer-focus:ring-2 peer-focus:ring-primary peer-focus:ring-offset-2',
            'peer-checked:bg-primary',
            'bg-gray-300 dark:bg-gray-600',
            disabled ? 'cursor-not-allowed' : 'cursor-pointer'
          )}
        />
        <div
          className={clsx(
            'absolute left-1 top-1 w-4 h-4 rounded-full transition-transform',
            'bg-white',
            'peer-checked:translate-x-5'
          )}
        />
      </div>
      {label && <span className="text-sm text-primary">{label}</span>}
    </label>
  )
}
