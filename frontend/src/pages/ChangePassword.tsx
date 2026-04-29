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

export function ChangePassword() {
  const { t } = useTranslation()
  const { clearMustChangePassword } = useAuthStore()
  const { addToast } = useToastStore()

  useMetaTags({
    title: t('auth.changePassword') || 'Change Password',
    description: 'Change your admin password',
  })

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setError('')

    if (newPassword !== confirmPassword) {
      setError(t('auth.passwordsDoNotMatch') || 'Passwords do not match')
      return
    }

    if (newPassword.length < 8) {
      setError(t('auth.passwordTooShort') || 'Password must be at least 8 characters')
      return
    }

    setIsLoading(true)

    try {
      await authApi.changePassword({
        current_password: currentPassword,
        new_password: newPassword,
      })

      clearMustChangePassword()
      addToast({
        type: 'success',
        message: t('auth.passwordChanged') || 'Password changed successfully',
      })

      route('/', true)
    } catch (err) {
      const axiosErr = err as AxiosError<{ message?: string; error?: string }>
      const errorMessage =
        axiosErr.response?.data?.error ||
        axiosErr.response?.data?.message ||
        t('auth.changePasswordError')
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
            <h1 className="text-2xl font-bold text-primary mb-2">
              {t('auth.changePassword') || 'Change Password'}
            </h1>
            <p className="text-sm text-secondary">
              {t('auth.changePasswordDescription') || 'Please change your password to continue'}
            </p>
          </div>

          {error && (
            <Alert variant="danger" className="mb-4">
              {error}
            </Alert>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              type="password"
              label={t('auth.currentPassword') || 'Current Password'}
              value={currentPassword}
              onChange={(e) => setCurrentPassword((e.target as HTMLInputElement).value)}
              placeholder="••••••••"
              required
              fullWidth
              disabled={isLoading}
            />
            <Input
              type="password"
              label={t('auth.newPassword') || 'New Password'}
              value={newPassword}
              onChange={(e) => setNewPassword((e.target as HTMLInputElement).value)}
              placeholder="••••••••"
              required
              fullWidth
              disabled={isLoading}
            />
            <Input
              type="password"
              label={t('auth.confirmPassword') || 'Confirm New Password'}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword((e.target as HTMLInputElement).value)}
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
              {t('auth.changePasswordButton') || 'Change Password'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}