import { Skeleton } from './Skeleton';

export function PageSkeleton() {
  return (
    <div className="space-y-6">
      {/* Page header skeleton */}
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <Skeleton className="h-8 w-48" variant="text" />
          <Skeleton className="h-4 w-64" variant="text" />
        </div>
        <Skeleton className="h-10 w-32" variant="rounded" />
      </div>
      
      {/* Stats cards skeleton */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4"
          >
            <Skeleton className="h-4 w-20 mb-2" variant="text" />
            <Skeleton className="h-8 w-24 mb-1" variant="text" />
            <Skeleton className="h-3 w-16" variant="text" />
          </div>
        ))}
      </div>
      
      {/* Content area skeleton */}
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <div className="border-b border-gray-200 dark:border-gray-700 p-4">
          <Skeleton className="h-5 w-32" variant="text" />
        </div>
        <div className="p-4 space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-4 w-full" variant="text" />
          ))}
        </div>
      </div>
    </div>
  );
}
