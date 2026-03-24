import { useQuery } from './useQuery'
import { useMutation } from './useMutation'
import { outboundApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'
import { cache } from '../utils/cache'
import i18n from '../i18n'
import type { Outbound } from '../types'

// List all outbounds
export function useOutbounds(params?: { core_id?: number; protocol?: string }) {
  return useQuery<Outbound[]>(
    `outbounds-${params?.core_id || 'all'}-${params?.protocol || 'all'}`,
    () => outboundApi.list(params).then((res) => res.data),
    { refetchInterval: 15000 }
  )
}

// Get single outbound
export function useOutbound(id: number) {
  return useQuery<Outbound>(
    `outbound-${id}`,
    () => outboundApi.get(id).then((res) => res.data),
    { enabled: !!id }
  )
}

// Create outbound mutation
export function useCreateOutbound() {
  const { addToast } = useToastStore()

  return useMutation(
    (data: Record<string, unknown>) => outboundApi.create(data).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('outbounds.outboundCreated') })
        cache.invalidatePattern(/^outbound/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Update outbound mutation
export function useUpdateOutbound() {
  const { addToast } = useToastStore()

  return useMutation(
    ({ id, data }: { id: number; data: Record<string, unknown> }) =>
      outboundApi.update(id, data).then((res) => res.data),
    {
      onSuccess: (_, { id }) => {
        addToast({ type: 'success', message: i18n.t('outbounds.outboundUpdated') })
        cache.invalidatePattern(/^outbound/)
        cache.invalidate(`outbound-${id}`)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Delete outbound mutation
export function useDeleteOutbound() {
  const { addToast } = useToastStore()

  return useMutation(
    (id: number) => outboundApi.delete(id).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('outbounds.outboundDeleted') })
        cache.invalidatePattern(/^outbound/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}
