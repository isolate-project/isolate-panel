import { useQuery } from './useQuery'
import { useMutation } from './useMutation'
import { inboundApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'
import { cache } from '../utils/cache'
import i18n from '../i18n'

// List all inbounds
export function useInbounds() {
  return useQuery(
    'inbounds',
    () => inboundApi.list().then((res) => res.data),
    { refetchInterval: 15000 }
  )
}

// Get single inbound
export function useInbound(id: number) {
  return useQuery(
    `inbound-${id}`,
    () => inboundApi.get(id).then((res) => res.data),
    { enabled: !!id }
  )
}

// Create inbound mutation
export function useCreateInbound() {
  const { addToast } = useToastStore()

  return useMutation(
    (data: Record<string, unknown>) => inboundApi.create(data).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('inbounds.inboundCreated') })
        cache.invalidatePattern(/^inbound/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Update inbound mutation
export function useUpdateInbound() {
  const { addToast } = useToastStore()

  return useMutation(
    ({ id, data }: { id: number; data: Record<string, unknown> }) =>
      inboundApi.update(id, data).then((res) => res.data),
    {
      onSuccess: (_, { id }) => {
        addToast({ type: 'success', message: i18n.t('inbounds.inboundUpdated') })
        cache.invalidatePattern(/^inbound/)
        cache.invalidate(`inbound-${id}`)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Delete inbound mutation
export function useDeleteInbound() {
  const { addToast } = useToastStore()

  return useMutation(
    (id: number) => inboundApi.delete(id).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('inbounds.inboundDeleted') })
        cache.invalidatePattern(/^inbound/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Assign user to inbound
export function useAssignUser() {
  const { addToast } = useToastStore()

  return useMutation(
    ({ inboundId, userId }: { inboundId: number; userId: number }) =>
      inboundApi.assign(inboundId, userId).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('inbounds.userAssigned') })
        cache.invalidatePattern(/^inbound/)
        cache.invalidatePattern(/^user/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Unassign user from inbound
export function useUnassignUser() {
  const { addToast } = useToastStore()

  return useMutation(
    ({ inboundId, userId }: { inboundId: number; userId: number }) =>
      inboundApi.unassign(inboundId, userId).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('inbounds.userUnassigned') })
        cache.invalidatePattern(/^inbound/)
        cache.invalidatePattern(/^user/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}
