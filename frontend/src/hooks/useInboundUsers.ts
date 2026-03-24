import { useQuery } from './useQuery'
import { useMutation } from './useMutation'
import { inboundApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'
import { cache } from '../utils/cache'
import i18n from '../i18n'
import type { User } from '../types'

// Get users assigned to an inbound
export function useInboundUsers(inboundId: number) {
  return useQuery<{ users: User[]; total: number }>(
    `inbound-users-${inboundId}`,
    () => inboundApi.getUsers(inboundId).then((res) => res.data),
    { enabled: !!inboundId }
  )
}

// Bulk assign/unassign users
export function useBulkAssignUsers() {
  const { addToast } = useToastStore()

  return useMutation(
    ({ inboundId, addUserIds, removeUserIds }: { inboundId: number; addUserIds: number[]; removeUserIds: number[] }) =>
      inboundApi.bulkAssignUsers(inboundId, addUserIds, removeUserIds).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('inbounds.bulkAssignSuccess') })
        cache.invalidatePattern(/^inbound/)
        cache.invalidatePattern(/^user/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}
