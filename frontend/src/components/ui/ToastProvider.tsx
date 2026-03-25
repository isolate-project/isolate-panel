import { Toaster } from 'sonner';
import type { JSX } from 'preact';

interface ToastProviderProps extends JSX.HTMLAttributes<HTMLDivElement> {
  position?:
    | 'top-left'
    | 'top-center'
    | 'top-right'
    | 'bottom-left'
    | 'bottom-center'
    | 'bottom-right';
  duration?: number;
  theme?: 'light' | 'dark' | 'system';
}

export function ToastProvider({
  position = 'top-right',
  duration = 4000,
  theme = 'system',
  ...props
}: ToastProviderProps) {
  return (
    <Toaster
      position={position}
      duration={duration}
      theme={theme}
      {...props}
      toastOptions={{
        classNames: {
          toast:
            'group toast group-[.toaster]:bg-white group-[.toaster]:text-gray-900 group-[.toaster]:border-gray-200 group-[.toaster]:shadow-lg dark:group-[.toaster]:bg-gray-800 dark:group-[.toaster]:text-gray-100 dark:group-[.toaster]:border-gray-700',
          description: 'group-[.toast]:text-gray-500 dark:group-[.toast]:text-gray-400',
          actionButton:
            'group-[.toast]:bg-blue-600 group-[.toast]:text-white group-[.toast]:hover:bg-blue-700',
          cancelButton:
            'group-[.toast]:bg-gray-200 group-[.toast]:text-gray-700 group-[.toast]:hover:bg-gray-300',
          success:
            'group-[.toaster]:border-l-4 group-[.toaster]:border-l-green-500',
          error:
            'group-[.toaster]:border-l-4 group-[.toaster]:border-l-red-500',
          warning:
            'group-[.toaster]:border-l-4 group-[.toaster]:border-l-yellow-500',
          info:
            'group-[.toaster]:border-l-4 group-[.toaster]:border-l-blue-500',
        },
      }}
    />
  );
}

export { toast } from 'sonner';
