import { useQuery } from './useQuery'
import { useMutation } from './useMutation'
import { subscriptionApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'
import { cache } from '../utils/cache'
import i18n from '../i18n'

export interface SubscriptionStats {
  total_accesses: number
  by_format: Record<string, number>
  by_day: Record<string, number>
  unique_ips: number
  last_access: string | null
}

export interface RegenerateResult {
  subscription_token: string
  subscription_url: string
  clash_url: string
  singbox_url: string
  qr_code_url: string
}

export function useSubscriptionStats(userId: number, days: number = 7) {
  return useQuery<SubscriptionStats>(
    `subscription-stats-${userId}-${days}`,
    () => subscriptionApi.getStats(userId, days).then(res => res.data as SubscriptionStats),
    { enabled: userId > 0 }
  )
}

export function useRegenerateToken(onSuccess?: () => void) {
  const { addToast } = useToastStore()

  return useMutation<RegenerateResult, { userId: number }>(
    ({ userId }) => subscriptionApi.regenerateToken(userId).then(res => res.data as RegenerateResult),
    {
      onSuccess: (_data, _variables) => {
        // Invalidate related queries
        cache.invalidatePattern(/subscription/)
        cache.invalidatePattern(/users/)
        
        addToast({
          type: 'success',
          message: i18n.t('subscriptions.regenerated'),
        })
        
        onSuccess?.()
      },
      onError: (error) => {
        addToast({
          type: 'error',
          message: error.message || i18n.t('errors.operationFailed'),
        })
      },
    }
  )
}

export function useSubscriptionURLs(_userId: number, subscriptionToken: string) {
  const baseUrl = window.location.origin
  
  return {
    v2ray: `${baseUrl}/sub/${subscriptionToken}`,
    clash: `${baseUrl}/sub/${subscriptionToken}/clash`,
    singbox: `${baseUrl}/sub/${subscriptionToken}/singbox`,
    qr: `${baseUrl}/sub/${subscriptionToken}/qr`,
  }
}
