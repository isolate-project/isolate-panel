import { useState } from 'preact/hooks'
import { route } from 'preact-router'

import { useTranslation } from 'react-i18next'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Card, CardContent } from '../components/ui/Card'
import { Alert } from '../components/ui/Alert'
import { useAuthStore } from '../stores/authStore'
import { useToastStore } from '../stores/toastStore'
import { useMetaTags } from '../hooks/useDocumentTitle'
import { authApi } from '../api/endpoints'
import { AxiosError } from 'axios'

export function Login() {
  const { t } = useTranslation()
  const { setTokens, setUser } = useAuthStore()
  const { addToast } = useToastStore()

  useMetaTags({
    title: t('auth.login') || 'Login',
    description: 'Sign in to Isolate Panel administration dashboard',
  })

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [requiresTotp, setRequirestotp] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      const response = await authApi.login(username, password, requiresTotp ? totpCode : undefined)
      const data = response.data as Record<string, unknown>

      // Backend signals that TOTP is required
      if (data.requires_totp) {
        setRequirestotp(true)
        setIsLoading(false)
        return
      }

      const { access_token, refresh_token, admin } = data as {
        access_token: string
        refresh_token: string
        admin: Parameters<typeof setUser>[0]
      }

      setTokens(access_token, refresh_token)
      setUser(admin)

      addToast({
        type: 'success',
        message: t('auth.welcome'),
      })

      route('/', true)
    } catch (err) {
      const axiosErr = err as AxiosError<{ message?: string; error?: string }>
      const errorMessage =
        axiosErr.response?.data?.error ||
        axiosErr.response?.data?.message ||
        t('auth.loginError')
      setError(errorMessage)
      addToast({ type: 'error', message: errorMessage })
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-secondary flex items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardContent className="p-6">
          <div className="text-center mb-6">
            <h1 className="text-2xl font-bold text-primary mb-2">{t('common.appName')}</h1>
            <p className="text-sm text-secondary">{t('auth.welcome')}</p>
          </div>

          {error && (
            <Alert variant="danger" className="mb-4">
              {error}
            </Alert>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            {!requiresTotp ? (
              <>
                <Input
                  type="text"
                  label={t('auth.username')}
                  value={username}
                  onChange={(e) => setUsername((e.target as HTMLInputElement).value)}
                  placeholder="admin"
                  required
                  fullWidth
                  disabled={isLoading}
                />
                <Input
                  type="password"
                  label={t('auth.password')}
                  value={password}
                  onChange={(e) => setPassword((e.target as HTMLInputElement).value)}
                  placeholder="••••••••"
                  required
                  fullWidth
                  disabled={isLoading}
                />
              </>
            ) : (
              <Input
                type="text"
                label={t('auth.totpCode') || 'Authenticator Code'}
                value={totpCode}
                onChange={(e) => setTotpCode((e.target as HTMLInputElement).value)}
                placeholder="000000"
                required
                fullWidth
                disabled={isLoading}
                autoFocus
              />
            )}

            <Button
              type="submit"
              variant="default"
              fullWidth
              loading={isLoading}
              disabled={isLoading}
            >
              {requiresTotp ? (t('auth.verify') || 'Verify') : t('auth.loginButton')}
            </Button>

            {requiresTotp && (
              <Button
                type="button"
                variant="ghost"
                fullWidth
                onClick={() => { setRequirestotp(false); setTotpCode(''); setError('') }}
              >
                {t('common.back') || 'Back'}
              </Button>
            )}
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
