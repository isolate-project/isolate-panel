import { useState, useCallback, useRef } from 'preact/hooks'

interface UseMutationOptions<TData, TVariables> {
  onSuccess?: (data: TData, variables: TVariables) => void
  onError?: (error: Error, variables: TVariables) => void
}

interface UseMutationResult<TData, TVariables> {
  mutate: (variables: TVariables) => Promise<TData | undefined>
  isLoading: boolean
  error: Error | null
  data: TData | null
  reset: () => void
}

export function useMutation<TData, TVariables>(
  mutationFn: (variables: TVariables) => Promise<TData>,
  options: UseMutationOptions<TData, TVariables> = {}
): UseMutationResult<TData, TVariables> {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const [data, setData] = useState<TData | null>(null)

  // Store mutationFn and options in refs to avoid dependency instability
  const mutationFnRef = useRef(mutationFn)
  mutationFnRef.current = mutationFn
  const onSuccessRef = useRef(options.onSuccess)
  onSuccessRef.current = options.onSuccess
  const onErrorRef = useRef(options.onError)
  onErrorRef.current = options.onError

  const mutate = useCallback(
    async (variables: TVariables) => {
      setIsLoading(true)
      setError(null)

      try {
        const result = await mutationFnRef.current(variables)
        setData(result)
        onSuccessRef.current?.(result, variables)
        return result
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Unknown error')
        setError(error)
        onErrorRef.current?.(error, variables)
        throw error
      } finally {
        setIsLoading(false)
      }
    },
    []
  )

  const reset = useCallback(() => {
    setIsLoading(false)
    setError(null)
    setData(null)
  }, [])

  return { mutate, isLoading, error, data, reset }
}
