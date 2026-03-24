import { useCallback } from 'preact/hooks'
import { cache } from '../utils/cache'

export function useOptimisticUpdate<T>(queryKey: string) {
  const optimisticUpdate = useCallback(
    async (
      updater: (oldData: T) => T,
      mutationFn: () => Promise<T>
    ): Promise<T> => {
      const oldData = cache.get<T>(queryKey)
      
      if (!oldData) {
        // No cached data, just run mutation
        return await mutationFn()
      }

      // Apply optimistic update
      const optimisticData = updater(oldData)
      cache.set(queryKey, optimisticData)

      try {
        // Run actual mutation
        const result = await mutationFn()
        cache.set(queryKey, result)
        return result
      } catch (error) {
        // Rollback on error
        cache.set(queryKey, oldData)
        throw error
      }
    },
    [queryKey]
  )

  return { optimisticUpdate }
}
