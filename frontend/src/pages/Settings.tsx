import { useState, useEffect } from 'preact/hooks'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Spinner } from '../components/ui/Spinner'
import { Alert } from '../components/ui/Alert'
import { useThemeStore } from '../stores/themeStore'
import { useToastStore } from '../stores/toastStore'
import { systemApi } from '../api/endpoints'
import { Alert as AlertBanner } from '../components/ui/Alert'
import { useTranslation } from 'react-i18next'
import { Moon, Sun, Globe, Save } from 'lucide-preact'

interface SettingsState {
  panel_name: string
  jwt_access_token_ttl: string
  jwt_refresh_token_ttl: string
  max_login_attempts: string
  log_level: string
}

interface MonitoringState {
  mode: 'lite' | 'full'
  interval: number
}

export function Settings() {
  const { t, i18n } = useTranslation()
  const { theme, setTheme } = useThemeStore()
  const { addToast } = useToastStore()

  const [settings, setSettings] = useState<SettingsState>({
    panel_name: 'Isolate Panel',
    jwt_access_token_ttl: '900',
    jwt_refresh_token_ttl: '604800',
    max_login_attempts: '5',
    log_level: 'info',
  })

  const [monitoring, setMonitoring] = useState<MonitoringState>({
    mode: 'lite',
    interval: 60,
  })

  const [isSaving, setIsSaving] = useState(false)
  const [isLoadingSettings, setIsLoadingSettings] = useState(true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // Load settings from backend on mount
  useEffect(() => {
    const loadSettings = async () => {
      try {
        const [settingsRes, monitoringRes] = await Promise.all([
          systemApi.getSettings(),
          systemApi.getMonitoring(),
        ])
        
        if (settingsRes.data) {
          const data = settingsRes.data
          setSettings({
            panel_name: data.panel_name || 'Isolate Panel',
            jwt_access_token_ttl: String(data.jwt_access_token_ttl || 900),
            jwt_refresh_token_ttl: String(data.jwt_refresh_token_ttl || 604800),
            max_login_attempts: String(data.max_login_attempts || 5),
            log_level: data.log_level || 'info',
          })
        }
        
        if (monitoringRes.data && monitoringRes.data.success) {
          setMonitoring({
            mode: monitoringRes.data.mode as 'lite' | 'full',
            interval: monitoringRes.data.interval || 60,
          })
        }
      } catch {
        // Backend may not be connected yet — use defaults
      } finally {
        setIsLoadingSettings(false)
      }
    }
    loadSettings()
  }, [])

  const handleLanguageChange = (e: Event) => {
    const target = e.target as HTMLSelectElement
    i18n.changeLanguage(target.value)
    addToast({ type: 'success', message: t('settings.languageChanged') })
  }

  const handleSettingChange = (e: Event) => {
    const target = e.target as HTMLInputElement | HTMLSelectElement
    const { name, value } = target
    setSettings(prev => ({ ...prev, [name]: value }))
    // Clear error on change
    if (errors[name]) {
      setErrors(prev => {
        const next = { ...prev }
        delete next[name]
        return next
      })
    }
  }

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {}
    if (!settings.panel_name.trim()) {
      newErrors.panel_name = t('errors.validationError')
    }
    const accessTTL = Number(settings.jwt_access_token_ttl)
    if (isNaN(accessTTL) || accessTTL < 60 || accessTTL > 86400) {
      newErrors.jwt_access_token_ttl = '60 - 86400'
    }
    const refreshTTL = Number(settings.jwt_refresh_token_ttl)
    if (isNaN(refreshTTL) || refreshTTL < 3600 || refreshTTL > 2592000) {
      newErrors.jwt_refresh_token_ttl = '3600 - 2592000'
    }
    const maxAttempts = Number(settings.max_login_attempts)
    if (isNaN(maxAttempts) || maxAttempts < 1 || maxAttempts > 100) {
      newErrors.max_login_attempts = '1 - 100'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSave = async () => {
    if (!validate()) return

    setIsSaving(true)
    try {
      await systemApi.updateSettings({
        panel_name: settings.panel_name,
        jwt_access_token_ttl: Number(settings.jwt_access_token_ttl),
        jwt_refresh_token_ttl: Number(settings.jwt_refresh_token_ttl),
        max_login_attempts: Number(settings.max_login_attempts),
        log_level: settings.log_level,
      })
      addToast({ type: 'success', message: t('settings.settingsSaved') })
    } catch {
      addToast({ type: 'error', message: t('errors.serverError') })
    } finally {
      setIsSaving(false)
    }
  }

  const handleMonitoringChange = async (e: Event) => {
    const target = e.target as HTMLSelectElement
    const newMode = target.value as 'lite' | 'full'
    
    try {
      await systemApi.updateMonitoring({ mode: newMode })
      setMonitoring({
        mode: newMode,
        interval: newMode === 'full' ? 10 : 60,
      })
      addToast({ type: 'success', message: t('settings.monitoringModeUpdated') })
    } catch {
      addToast({ type: 'error', message: t('errors.serverError') })
    }
  }

  return (
    <PageLayout>
      <PageHeader
        title={t('settings.title')}
        description={t('settings.description')}
      />

      {isLoadingSettings ? (
        <Card className="flex items-center justify-center py-12">
      <CardContent className="p-6">
          <Spinner size="lg" />
              </CardContent>
    </Card>
      ) : (
        <div className="space-y-6">
          {/* Appearance Settings */}
          <Card>
      <CardContent className="p-6">
            <h3 className="text-lg font-semibold text-primary mb-4">
              {t('settings.appearance')}
            </h3>

            <div className="space-y-4">
              {/* Theme Selector */}
              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  {t('settings.theme')}
                </label>
                <div className="flex gap-3">
                  <Button
                    variant={theme === 'light' ? 'primary' : 'secondary'}
                    onClick={() => setTheme('light')}
                    className="flex-1"
                  >
                    <Sun className="w-4 h-4 mr-2" />
                    {t('settings.lightMode')}
                  </Button>
                  <Button
                    variant={theme === 'dark' ? 'primary' : 'secondary'}
                    onClick={() => setTheme('dark')}
                    className="flex-1"
                  >
                    <Moon className="w-4 h-4 mr-2" />
                    {t('settings.darkMode')}
                  </Button>
                </div>
              </div>

              {/* Language Selector */}
              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  <Globe className="w-4 h-4 inline mr-2" />
                  {t('settings.language')}
                </label>
                <Select
                  value={i18n.language}
                  onChange={handleLanguageChange}
                  options={[
                    { value: 'en', label: 'English' },
                    { value: 'ru', label: 'Русский' },
                    { value: 'zh', label: '中文' },
                  ]}
                  fullWidth
                />
              </div>
            </div>
                </CardContent>
    </Card>

          {/* General Settings */}
          <Card>
      <CardContent className="p-6">
            <h3 className="text-lg font-semibold text-primary mb-4">
              {t('settings.general')}
            </h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  {t('settings.panelName')}
                </label>
                <Input
                  name="panel_name"
                  type="text"
                  value={settings.panel_name}
                  onChange={handleSettingChange}
                  placeholder={t('common.appName')}
                  error={errors.panel_name}
                  fullWidth
                />
              </div>
            </div>
                </CardContent>
    </Card>

          {/* Monitoring Mode Settings */}
          <Card>
      <CardContent className="p-6">
            <h3 className="text-lg font-semibold text-primary mb-4">
              {t('settings.monitoringMode')}
            </h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  {t('settings.monitoringModeLabel')}
                </label>
                <Select
                  name="monitoring_mode"
                  value={monitoring.mode}
                  onChange={handleMonitoringChange}
                  options={[
                    { value: 'lite', label: t('settings.monitoringModeLite') },
                    { value: 'full', label: t('settings.monitoringModeFull') },
                  ]}
                  fullWidth
                />
                <div className="mt-3 p-3 bg-secondary/50 rounded-lg">
                  <AlertBanner variant="info" className="text-sm">
                    {monitoring.mode === 'lite' 
                      ? t('settings.monitoringModeLiteDesc')
                      : t('settings.monitoringModeFullDesc')
                    }
                  </AlertBanner>
                </div>
              </div>

              <div className="pt-2">
                <div className="flex justify-between text-sm">
                  <span className="text-tertiary">{t('settings.currentInterval')}</span>
                  <span className="font-medium text-primary">
                    {monitoring.interval} {t('settings.seconds')}
                  </span>
                </div>
              </div>
            </div>
                </CardContent>
    </Card>

          {/* Security Settings */}
          <Card>
      <CardContent className="p-6">
            <h3 className="text-lg font-semibold text-primary mb-4">
              {t('settings.security')}
            </h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  {t('settings.jwtTokenTTL')} ({t('settings.seconds')})
                </label>
                <Input
                  name="jwt_access_token_ttl"
                  type="number"
                  value={settings.jwt_access_token_ttl}
                  onChange={handleSettingChange}
                  placeholder="900"
                  error={errors.jwt_access_token_ttl}
                  fullWidth
                />
                <p className="mt-1 text-xs text-tertiary">
                  {t('settings.jwtTokenTTLHint')}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  {t('settings.refreshTokenTTL')} ({t('settings.seconds')})
                </label>
                <Input
                  name="jwt_refresh_token_ttl"
                  type="number"
                  value={settings.jwt_refresh_token_ttl}
                  onChange={handleSettingChange}
                  placeholder="604800"
                  error={errors.jwt_refresh_token_ttl}
                  fullWidth
                />
                <p className="mt-1 text-xs text-tertiary">
                  {t('settings.refreshTokenTTLHint')}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  {t('settings.maxLoginAttempts')}
                </label>
                <Input
                  name="max_login_attempts"
                  type="number"
                  value={settings.max_login_attempts}
                  onChange={handleSettingChange}
                  placeholder="5"
                  error={errors.max_login_attempts}
                  fullWidth
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-primary mb-2">
                  {t('settings.logLevel')}
                </label>
                <Select
                  name="log_level"
                  value={settings.log_level}
                  onChange={handleSettingChange}
                  options={[
                    { value: 'debug', label: t('settings.logLevelDebug') },
                    { value: 'info', label: t('settings.logLevelInfo') },
                    { value: 'warn', label: t('settings.logLevelWarning') },
                    { value: 'error', label: t('settings.logLevelError') },
                  ]}
                  fullWidth
                />
              </div>
            </div>
                </CardContent>
    </Card>

          {/* Info Alert */}
          <Alert variant="info">
            {t('settings.backendNote')}
          </Alert>

          {/* Save Button */}
          <div className="flex justify-end">
            <Button variant="default" onClick={handleSave} disabled={isSaving}>
              {isSaving ? (
                <><Spinner size="sm" className="mr-2" />{t('common.loading')}</>
              ) : (
                <><Save className="w-4 h-4 mr-2" />{t('common.save')}</>
              )}
            </Button>
          </div>
        </div>
      )}
    </PageLayout>
  )
}
