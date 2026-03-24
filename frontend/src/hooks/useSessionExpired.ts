import { useEffect } from 'preact/hooks'
import { useToastStore } from '../stores/toastStore'
import i18n from '../i18n'

let hasShownSessionExpired = false

export function useSessionExpired() {
  const { addToast } = useToastStore()

  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      // Detect when tokens are removed (logout or session expired)
      if (e.key === 'accessToken' && e.oldValue && !e.newValue) {
        if (!hasShownSessionExpired && window.location.pathname !== '/login') {
          hasShownSessionExpired = true
          addToast({
            type: 'warning',
            message: i18n.t('auth.sessionExpired'),
            duration: 5000,
          })
          setTimeout(() => {
            hasShownSessionExpired = false
          }, 6000)
        }
      }
    }

    window.addEventListener('storage', handleStorageChange)
    return () => window.removeEventListener('storage', handleStorageChange)
  }, [addToast])
}
