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
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      const response = await authApi.login(username, password)
      const { access_token, refresh_token, admin } = response.data

      setTokens(access_token, refresh_token)
      setUser(admin)
      
      addToast({
        type: 'success',
        message: t('auth.welcome'),
      })

      // Wait for localStorage to be updated before redirect
      await new Promise(resolve => setTimeout(resolve, 100))
      
      // SPA navigation
      route('/', true)
    } catch (err) {
      const axiosErr = err as AxiosError<{ message?: string }>
      const errorMessage = axiosErr.response?.data?.message || t('auth.loginError')
      setError(errorMessage)
      addToast({
        type: 'error',
        message: errorMessage,
      })
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-secondary flex items-center justify-center p-4">
      <Card className="w-full max-w-md" >
      <CardContent className="p-6">
        <div className="text-center mb-6">
          <h1 className="text-2xl font-bold text-primary mb-2">{t('common.appName')}</h1>
          <p className="text-sm text-secondary">
            {t('auth.welcome')}
          </p>
        </div>

        {error && (
          <Alert variant="danger" className="mb-4">
            {error}
          </Alert>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
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

          <Button
            type="submit"
            variant="default"
            fullWidth
            loading={isLoading}
            disabled={isLoading}
          >
            {t('auth.loginButton')}
          </Button>
        </form>
            </CardContent>
    </Card>
    </div>
  )
}
