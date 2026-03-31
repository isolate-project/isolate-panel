import { useEffect, useRef } from 'preact/hooks'
import { useToastStore } from '../stores/toastStore'
import i18n from '../i18n'

let hasShownSessionExpired = false

export function resetSessionExpiredFlag_FOR_TESTING() {
  hasShownSessionExpired = false
}

export function useSessionExpired() {
  const { addToast } = useToastStore()

  const timeoutRef = useRef<number | null>(null)

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
          
          if (timeoutRef.current) window.clearTimeout(timeoutRef.current)
          
          timeoutRef.current = window.setTimeout(() => {
            hasShownSessionExpired = false
            timeoutRef.current = null
          }, 6000)
        }
      }
    }

    window.addEventListener('storage', handleStorageChange)
    return () => {
      window.removeEventListener('storage', handleStorageChange)
      if (timeoutRef.current) window.clearTimeout(timeoutRef.current)
    }
  }, [addToast])
}
