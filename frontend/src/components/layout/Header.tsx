import { Menu, Moon, Sun, Globe, LogOut, Cpu, HardDrive } from 'lucide-preact'
import { route } from 'preact-router'
import { useThemeStore } from '../../stores/themeStore'
import { useAuthStore } from '../../stores/authStore'
import { useSystemResources } from '../../hooks/useSystem'
import { invalidateAuthCache } from '../../router/ProtectedRoute'
import { useTranslation } from 'react-i18next'

interface HeaderProps {
  onMenuClick: () => void
}

export function Header({ onMenuClick }: HeaderProps) {
  const { theme, toggleTheme } = useThemeStore()
  const { user, logout } = useAuthStore()
  const { data: resources } = useSystemResources()
  const { t, i18n } = useTranslation()

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng)
  }

  const handleLogout = () => {
    invalidateAuthCache()
    logout()
    route('/login', true)
  }

  const ramPercent = resources?.ram?.percent || 0
  const cpuPercent = resources?.cpu?.percent || 0

  return (
    <header className="h-16 bg-primary border-b border-primary sticky top-0 z-sticky">
      <div className="h-full px-4 flex items-center justify-between">
        {/* Left: Mobile menu button */}
        <button
          onClick={onMenuClick}
          className="lg:hidden p-2 hover:bg-hover rounded-lg transition-base"
          aria-label="Toggle menu"
        >
          <Menu className="w-6 h-6" />
        </button>

        {/* Center: System Metrics Widget */}
        <div className="flex-1 flex justify-center">
          {resources && (
            <div className="hidden md:flex items-center gap-4 text-sm">
              <div className="flex items-center gap-2">
                <HardDrive className="w-4 h-4 text-tertiary" />
                <span className="text-secondary">{t('dashboard.ramUsage')}</span>
                <span className={
                  ramPercent > 85 ? 'text-danger font-semibold' :
                  ramPercent > 70 ? 'text-warning font-semibold' :
                  'text-success'
                }>
                  {ramPercent}%
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Cpu className="w-4 h-4 text-tertiary" />
                <span className="text-secondary">{t('dashboard.cpuUsage')}</span>
                <span className="text-primary font-medium">
                  {cpuPercent}%
                </span>
              </div>
            </div>
          )}
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-2">
          {/* Theme Toggle */}
          <button
            onClick={toggleTheme}
            className="p-2 hover:bg-hover rounded-lg transition-base"
            aria-label="Toggle theme"
          >
            {theme === 'light' ? (
              <Moon className="w-5 h-5" />
            ) : (
              <Sun className="w-5 h-5" />
            )}
          </button>

          {/* Language Switcher */}
          <div className="relative group">
            <button
              className="p-2 hover:bg-hover rounded-lg transition-base"
              aria-label="Change language"
            >
              <Globe className="w-5 h-5" />
            </button>
            <div className="absolute right-0 mt-2 w-32 bg-primary border border-primary rounded-lg shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-base">
              <button
                onClick={() => changeLanguage('en')}
                className="block w-full px-4 py-2 text-left text-sm hover:bg-hover transition-base"
              >
                English
              </button>
              <button
                onClick={() => changeLanguage('ru')}
                className="block w-full px-4 py-2 text-left text-sm hover:bg-hover transition-base"
              >
                Русский
              </button>
              <button
                onClick={() => changeLanguage('zh')}
                className="block w-full px-4 py-2 text-left text-sm hover:bg-hover transition-base"
              >
                中文
              </button>
            </div>
          </div>

          {/* User Menu */}
          {user && (
            <div className="relative group">
              <button className="flex items-center gap-2 px-3 py-2 hover:bg-hover rounded-lg transition-base">
                <div className="w-8 h-8 rounded-full bg-primary text-white flex items-center justify-center text-sm font-medium">
                  {user.username.charAt(0).toUpperCase()}
                </div>
                <span className="hidden md:block text-sm font-medium">
                  {user.username}
                </span>
              </button>
              <div className="absolute right-0 mt-2 w-48 bg-primary border border-primary rounded-lg shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-base">
                <div className="px-4 py-3 border-b border-primary">
                  <p className="text-sm font-medium">{user.username}</p>
                  <p className="text-xs text-secondary">
                    {user.is_super_admin ? t('auth.superAdmin') : t('auth.admin')}
                  </p>
                </div>
                <button
                  onClick={handleLogout}
                  className="flex items-center gap-2 w-full px-4 py-2 text-left text-sm hover:bg-hover transition-base text-danger"
                >
                  <LogOut className="w-4 h-4" />
                  {t('auth.logout')}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
