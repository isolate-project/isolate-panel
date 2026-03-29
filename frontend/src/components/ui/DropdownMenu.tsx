import { ComponentProps, useState, useEffect, useRef } from 'preact/compat'
import { cn } from '../../lib/utils'

export function DropdownMenu({ children }: { children: preact.ComponentChildren }) {
  return <div className="relative inline-block text-left">{children}</div>
}

export function DropdownMenuTrigger({ 
  children, 
  onClick 
}: { 
  children: preact.ComponentChildren, 
  onClick?: () => void 
}) {
  return <div onClick={onClick} className="cursor-pointer">{children}</div>
}

export function DropdownMenuContent({ 
  children, 
  isOpen, 
  onClose,
  align = 'right'
}: { 
  children: preact.ComponentChildren, 
  isOpen: boolean, 
  onClose: () => void,
  align?: 'left' | 'right'
}) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        onClose()
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div
      ref={ref}
      className={cn(
        "absolute z-dropdown mt-2 w-56 rounded-xl border border-border-primary bg-bg-primary p-1 shadow-glass transition-all animate-in fade-in zoom-in-95",
        align === 'right' ? 'right-0 origin-top-right' : 'left-0 origin-top-left'
      )}
    >
      {children}
    </div>
  )
}

export function DropdownMenuItem({ 
  children, 
  onClick, 
  variant = 'default',
  className
}: { 
  children: preact.ComponentChildren, 
  onClick?: () => void,
  variant?: 'default' | 'danger',
  className?: string
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex w-full cursor-pointer items-center rounded-md px-3 py-2 text-sm outline-none transition-colors",
        variant === 'danger' 
          ? "text-color-danger hover:bg-color-danger/10 hover:text-color-danger" 
          : "text-text-primary hover:bg-bg-hover",
        className
      )}
    >
      {children}
    </button>
  )
}

export function DropdownMenuSeparator() {
  return <div className="my-1 h-px bg-border-primary" />
}
