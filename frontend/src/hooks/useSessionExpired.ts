import { useEffect, useRef } from 'preact/hooks'
import { useToastStore } from '../stores/toastStore'
import i18n from '../i18n'

export function useSessionExpired() {
  const { addToast } = useToastStore()

  const hasShownRef = useRef(false)
  const timeoutRef = useRef<number | null>(null)

  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      // Detect when tokens are removed (logout or session expired)
      if (e.key === 'accessToken' && e.oldValue && !e.newValue) {
        if (!hasShownRef.current && window.location.pathname !== '/login') {
          hasShownRef.current = true
          addToast({
            type: 'warning',
            message: i18n.t('auth.sessionExpired'),
            duration: 5000,
          })

          if (timeoutRef.current) window.clearTimeout(timeoutRef.current)

          timeoutRef.current = window.setTimeout(() => {
            hasShownRef.current = false
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
