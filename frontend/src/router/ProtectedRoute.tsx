import { ComponentChildren } from 'preact'
import { useEffect, useState, useRef } from 'preact/hooks'
import { route } from 'preact-router'
import { useAuthStore } from '../stores/authStore'
import { Spinner } from '../components/ui/Spinner'
import { authApi } from '../api/endpoints'

interface ProtectedRouteProps {
  children: ComponentChildren
}

// Module-level cache (persists across renders)
let authVerified = false
let authVerifiedAt = 0
const AUTH_CACHE_TTL = 60000 // 1 minute

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { isAuthenticated, setUser, logout } = useAuthStore()
  const accessToken = useAuthStore(s => s.accessToken)
  const [isChecking, setIsChecking] = useState(true)
  const abortControllerRef = useRef<AbortController | null>(null)

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [])

  useEffect(() => {
    const checkAuth = async () => {
      // Abort any previous request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      abortControllerRef.current = new AbortController()

      const token = localStorage.getItem('accessToken')

      // No token, redirect to login
      if (!token) {
        setIsChecking(false)
        route('/login', true)
        return
      }

      // If recently verified, skip API call
      if (authVerified && Date.now() - authVerifiedAt < AUTH_CACHE_TTL && isAuthenticated) {
        setIsChecking(false)
        return
      }

      // Token exists, verify it's valid by fetching user info
      try {
        const response = await authApi.me()

        // Don't update if aborted
        if (abortControllerRef.current?.signal.aborted) {
          return
        }

        setUser(response.data)
        authVerified = true
        authVerifiedAt = Date.now()
        setIsChecking(false)
      } catch (err) {
        // Ignore abort errors
        if (err instanceof Error && err.name === 'AbortError') {
          return
        }

        // Token invalid or expired, logout and redirect
        authVerified = false
        logout()
        route('/login', true)
      }
    }

    checkAuth()
  }, [accessToken])  // Re-run when token changes (login/logout)

  // Show loading spinner while checking authentication
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

  // If not authenticated after check, don't render (redirect will happen)
  if (!isAuthenticated) {
    return null
  }

  // Authenticated, render protected content
  return <>{children}</>
}

// Export a function to invalidate auth cache (used on logout)
export function invalidateAuthCache() {
  authVerified = false
  authVerifiedAt = 0
}
