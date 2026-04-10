import { cn } from '../../lib/utils'

interface SliderProps {
  value: number
  onChange: (value: number) => void
  min?: number
  max?: number
  step?: number
  disabled?: boolean
  className?: string
  formatLabel?: (value: number) => string
}

export function Slider({
  value,
  onChange,
  min = 0,
  max = 100,
  step = 1,
  disabled = false,
  className,
  formatLabel,
}: SliderProps) {
  const percentage = ((value - min) / (max - min)) * 100

  return (
    <div className={cn('space-y-2', className)}>
      <div className="flex items-center gap-3">
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(Number((e.target as HTMLInputElement).value))}
          className={cn(
            'slider-input w-full h-2 rounded-full appearance-none cursor-pointer',
            'bg-bg-tertiary',
            'disabled:opacity-50 disabled:cursor-not-allowed',
          )}
          style={{
            background: `linear-gradient(to right, var(--color-primary) 0%, var(--color-primary) ${percentage}%, var(--bg-tertiary) ${percentage}%, var(--bg-tertiary) 100%)`,
          }}
        />
        <span className="text-sm font-medium text-text-primary tabular-nums min-w-[4ch] text-right">
          {formatLabel ? formatLabel(value) : value}
        </span>
      </div>
      <div className="flex justify-between text-xs text-text-tertiary">
        <span>{formatLabel ? formatLabel(min) : min}</span>
        <span>{formatLabel ? formatLabel(max) : max}</span>
      </div>
    </div>
  )
}
