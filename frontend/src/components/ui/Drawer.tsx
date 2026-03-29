import { ComponentProps, useEffect, useRef } from 'preact/compat'
import { createPortal } from 'preact/compat'
import { cn } from '../../lib/utils'
import { X } from 'lucide-preact'
import { Button } from './Button'

interface DrawerProps {
  isOpen: boolean
  onClose: () => void
  title?: string
  description?: string
  children: preact.ComponentChildren
  footer?: preact.ComponentChildren
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  className?: string
}

export function Drawer({
  isOpen,
  onClose,
  title,
  description,
  children,
  footer,
  size = 'md',
  className
}: DrawerProps) {
  const overlayRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) onClose()
    }
    document.addEventListener('keydown', handleEscape)
    
    if (isOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = 'unset'
    }

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.body.style.overflow = 'unset'
    }
  }, [isOpen, onClose])

  if (!isOpen) return null

  const sizeClasses = {
    sm: 'w-full sm:w-[400px]',
    md: 'w-full sm:w-[500px]',
    lg: 'w-full sm:w-[600px] md:w-[700px]',
    xl: 'w-full sm:w-[800px] md:w-[900px]',
    full: 'w-full'
  }

  const drawerContent = (
    <div className="fixed inset-0 z-modal flex justify-end">
      <div 
        ref={overlayRef}
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      <div 
        className={cn(
          "relative flex h-full flex-col bg-bg-secondary border-l border-border-primary shadow-2xl",
          sizeClasses[size],
          className
        )}
      >
        <div className="flex items-center justify-between border-b border-border-primary px-6 py-4 bg-bg-secondary">
          <div className="flex flex-col gap-1">
            {title && <h2 className="text-lg font-semibold text-text-primary">{title}</h2>}
            {description && <p className="text-sm text-text-secondary">{description}</p>}
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} className="shrink-0 rounded-full h-8 w-8 ml-4">
            <X className="h-4 w-4 text-text-secondary" />
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-6 scrollbar-thin scrollbar-thumb-border-secondary bg-bg-secondary">
          {children}
        </div>

        {footer && (
          <div className="flex items-center justify-end gap-3 border-t border-border-primary bg-bg-secondary px-6 py-4">
            {footer}
          </div>
        )}
      </div>
    </div>
  )

  return createPortal(drawerContent, document.body)
}
