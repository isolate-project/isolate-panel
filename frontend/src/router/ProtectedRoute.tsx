import { ComponentChildren } from 'preact'
import { useEffect, useState, useRef } from 'preact/hooks'
import { route } from 'preact-router'
import { useAuthStore } from '../stores/authStore'
import { Spinner } from '../components/ui/Spinner'
import { authApi } from '../api/endpoints'

interface ProtectedRouteProps {
  children: ComponentChildren
}

let authVerified = false
let authVerifiedAt = 0
const AUTH_CACHE_TTL = 60000

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { isAuthenticated, setUser, logout } = useAuthStore()
  const user = useAuthStore(s => s.user)
  const [isChecking, setIsChecking] = useState(true)
  const abortControllerRef = useRef<AbortController | null>(null)

  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [])

  useEffect(() => {
    const checkAuth = async () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      abortControllerRef.current = new AbortController()

      if (authVerified && Date.now() - authVerifiedAt < AUTH_CACHE_TTL && isAuthenticated) {
        setIsChecking(false)
        return
      }

      try {
        const response = await authApi.me()

        if (abortControllerRef.current?.signal.aborted) {
          return
        }

        setUser(response.data)
        authVerified = true
        authVerifiedAt = Date.now()
        setIsChecking(false)
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') {
          return
        }

        authVerified = false
        logout()
        route('/login', true)
      }
    }

    checkAuth()
  }, [isAuthenticated])

  if (isChecking) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-secondary">
        <div className="text-center">
          <Spinner size="lg" />
          <p className="mt-4 text-sm text-secondary">Verifying authentication...</p>
        </div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return null
  }

  if (user?.must_change_password && typeof window !== 'undefined' && !window.location.pathname.endsWith('/change-password')) {
    route('/change-password', true)
    return null
  }

  return <>{children}</>
}

export function invalidateAuthCache() {
  authVerified = false
  authVerifiedAt = 0
}
