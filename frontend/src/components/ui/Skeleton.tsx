import { clsx } from 'clsx';

interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular' | 'rounded';
  width?: string | number;
  height?: string | number;
  animation?: 'pulse' | 'wave' | false;
}

export function Skeleton({
  className,
  variant = 'text',
  width,
  height,
  animation = 'pulse',
}: SkeletonProps) {
  const baseStyles = 'bg-gray-200 dark:bg-gray-700';
  
  const variantStyles = {
    text: 'rounded',
    circular: 'rounded-full',
    rectangular: 'rounded-none',
    rounded: 'rounded-lg',
  };

  const animationStyles = {
    pulse: 'animate-pulse',
    wave: 'animate-shimmer',
    false: '',
  };

  const style: Record<string, string | number> = {};
  if (width !== undefined) {
    style.width = typeof width === 'string' ? width : `${width}px`;
  }
  if (height !== undefined) {
    style.height = typeof height === 'string' ? height : `${height}px`;
  }

  return (
    <div
      className={clsx(
        baseStyles,
        variantStyles[variant],
        animationStyles[animation],
        className
      )}
      style={Object.keys(style).length > 0 ? style : undefined}
      aria-hidden="true"
    />
  );
}
