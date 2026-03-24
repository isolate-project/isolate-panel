import { useQuery } from './useQuery'
import { useMutation } from './useMutation'
import { userApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'
import { cache } from '../utils/cache'
import i18n from '../i18n'

// List all users with pagination
export function useUsers(params?: { page?: number; limit?: number }) {
  return useQuery(
    `users-${JSON.stringify(params || {})}`,
    () => userApi.list(params).then((res) => res.data),
    {
      refetchInterval: 30000, // Refetch every 30 seconds
    }
  )
}

// Get user's inbounds
export function useUserInbounds(userId: number) {
  return useQuery(
    `user-${userId}-inbounds`,
    () => userApi.getInbounds(userId).then((res) => res.data),
    { enabled: !!userId }
  )
}

// Get single user
export function useUser(id: number) {
  return useQuery(
    `user-${id}`,
    () => userApi.get(id).then((res) => res.data),
    { enabled: !!id }
  )
}

// Create user mutation
export function useCreateUser() {
  const { addToast } = useToastStore()

  return useMutation(
    (data: Record<string, unknown>) => userApi.create(data).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('users.userCreated') })
        cache.invalidatePattern(/^users-/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Update user mutation
export function useUpdateUser() {
  const { addToast } = useToastStore()

  return useMutation(
    ({ id, data }: { id: number; data: Record<string, unknown> }) =>
      userApi.update(id, data).then((res) => res.data),
    {
      onSuccess: (_, { id }) => {
        addToast({ type: 'success', message: i18n.t('users.userUpdated') })
        cache.invalidatePattern(/^users-/)
        cache.invalidate(`user-${id}`)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Delete user mutation
export function useDeleteUser() {
  const { addToast } = useToastStore()

  return useMutation(
    (id: number) => userApi.delete(id).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: i18n.t('users.userDeleted') })
        cache.invalidatePattern(/^users-/)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

// Regenerate user credentials mutation
export function useRegenerateCredentials() {
  const { addToast } = useToastStore()

  return useMutation(
    (id: number) => userApi.regenerate(id).then((res) => res.data),
    {
      onSuccess: (_, id) => {
        addToast({ type: 'success', message: i18n.t('users.credentialsRegenerated') })
        cache.invalidate(`user-${id}`)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}
