import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';
import 'tippy.js/themes/light.css';
import 'tippy.js/animations/shift-away.css';
import { clsx } from 'clsx';
import type { ComponentChildren } from 'preact';

interface TooltipProps {
  children: ComponentChildren;
  content: string | ComponentChildren;
  position?: 'top' | 'bottom' | 'left' | 'right';
  variant?: 'default' | 'info' | 'warning' | 'error';
  maxWidth?: number;
  className?: string;
  delay?: number;
}

export function Tooltip({
  children,
  content,
  position = 'top',
  variant = 'default',
  maxWidth = 300,
  className,
  delay = 200,
}: TooltipProps) {
  const variantStyles = {
    default: '',
    info: 'tooltip-info',
    warning: 'tooltip-warning',
    error: 'tooltip-error',
  };

  return (
    <Tippy
      content={content}
      placement={position}
      animation="shift-away"
      theme="light"
      delay={delay}
      maxWidth={maxWidth}
      className={clsx(variantStyles[variant], className)}
      arrow={true}
      interactive={true}
    >
      {children}
    </Tippy>
  );
}

interface TooltipIconProps {
  content: string;
  position?: 'top' | 'bottom' | 'left' | 'right';
  size?: 'sm' | 'md' | 'lg';
}

export function TooltipIcon({ content, position = 'top', size = 'sm' }: TooltipIconProps) {
  const sizeClasses = {
    sm: 'w-4 h-4',
    md: 'w-5 h-5',
    lg: 'w-6 h-6',
  };

  return (
    <Tooltip content={content} position={position}>
      <span className="inline-flex items-center justify-center">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={clsx('text-gray-400 dark:text-gray-500', sizeClasses[size])}
        >
          <circle cx="12" cy="12" r="10" />
          <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
          <path d="M12 17h.01" />
        </svg>
      </span>
    </Tooltip>
  );
}
