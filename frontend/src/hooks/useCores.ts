import { useQuery } from './useQuery'
import { useMutation } from './useMutation'
import { coreApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'
import { cache } from '../utils/cache'
import i18n from '../i18n'

// Stable fetcher function (not recreated on each render)
const fetchCoresList = () => coreApi.list().then((res) => res.data)

// List all cores - NO POLLING for now
export function useCores() {
  return useQuery('cores', fetchCoresList, {
    refetchInterval: undefined, // Disabled polling
  })
}

// Get single core
export function useCore(name: string) {
  return useQuery(
    `core-${name}`,
    () => coreApi.get(name).then((res) => res.data),
    { enabled: !!name }
  )
}

// Get core status - NO POLLING for now
export function useCoreStatus(name: string) {
  return useQuery(
    `core-status-${name}`,
    () => coreApi.status(name).then((res) => res.data),
    {
      enabled: !!name,
      refetchInterval: undefined, // Disabled polling
    }
  )
}

// Start core mutation
export function useStartCore() {
  const { addToast } = useToastStore()

  return useMutation(
    (name: string) => coreApi.start(name).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('cores.coreStarted') })
        cache.invalidatePattern(/^core/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Stop core mutation
export function useStopCore() {
  const { addToast } = useToastStore()

  return useMutation(
    (name: string) => coreApi.stop(name).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('cores.coreStopped') })
        cache.invalidatePattern(/^core/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Restart core mutation
export function useRestartCore() {
  const { addToast } = useToastStore()

  return useMutation(
    (name: string) => coreApi.restart(name).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('cores.coreRestarted') })
        cache.invalidatePattern(/^core/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}
