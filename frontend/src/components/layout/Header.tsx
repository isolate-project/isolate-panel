import { Menu, Moon, Sun, Globe, LogOut, Cpu, HardDrive } from 'lucide-preact'
import { route } from 'preact-router'
import { useThemeStore } from '../../stores/themeStore'
import { useAuthStore } from '../../stores/authStore'
import { useSystemResources } from '../../hooks/useSystem'
import { invalidateAuthCache } from '../../router/ProtectedRoute'
import { useTranslation } from 'react-i18next'
import { Button } from '../ui/Button'
import { cn } from '../../lib/utils'

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
    <header className="sticky top-0 z-sticky h-16 border-b border-border-primary bg-bg-primary/80 backdrop-blur-md">
      <div className="flex h-full items-center justify-between px-4 sm:px-6">
        {/* Left: Mobile menu button & Breadcrumbs placeholder */}
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={onMenuClick}
            className="lg:hidden"
            aria-label="Toggle menu"
          >
            <Menu className="h-5 w-5" />
          </Button>
        </div>

        {/* Center: System Metrics Widget */}
        <div className="flex-1 flex justify-center">
          {resources && (
            <div className="hidden md:flex items-center gap-6 rounded-full border border-border-primary bg-bg-secondary/50 px-4 py-1.5 shadow-sm">
              <div className="flex items-center gap-2 text-sm font-medium">
                <HardDrive className="h-4 w-4 text-text-tertiary" />
                <span className={cn(
                  "transition-colors",
                  ramPercent > 85 ? 'text-color-danger' :
                  ramPercent > 70 ? 'text-color-warning' :
                  'text-text-primary'
                )}>
                  RAM {ramPercent}%
                </span>
              </div>
              <div className="h-4 w-px bg-border-primary"></div>
              <div className="flex items-center gap-2 text-sm font-medium">
                <Cpu className="h-4 w-4 text-text-tertiary" />
                <span className={cn(
                  "transition-colors",
                  cpuPercent > 80 ? 'text-color-danger' : 
                  'text-text-primary'
                )}>
                  CPU {cpuPercent}%
                </span>
              </div>
            </div>
          )}
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-2">
          {/* Theme Toggle */}
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            aria-label="Toggle theme"
            className="text-text-secondary hover:text-text-primary"
          >
            {theme === 'light' ? (
              <Moon className="h-5 w-5" />
            ) : (
              <Sun className="h-5 w-5" />
            )}
          </Button>

          {/* Language Switcher */}
          <div className="group relative">
            <Button
              variant="ghost"
              size="icon"
              aria-label="Change language"
              className="text-text-secondary hover:text-text-primary"
            >
              <Globe className="h-5 w-5" />
            </Button>
            <div className="absolute right-0 mt-1 w-32 origin-top-right rounded-xl border border-border-primary bg-bg-primary p-1 shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
              <button
                onClick={() => changeLanguage('en')}
                className={cn(
                  "block w-full rounded-md px-3 py-2 text-left text-sm transition-colors",
                  i18n.language === 'en' ? "bg-color-primary/10 text-color-primary font-medium" : "hover:bg-bg-hover text-text-secondary hover:text-text-primary"
                )}
              >
                English
              </button>
              <button
                onClick={() => changeLanguage('ru')}
                className={cn(
                  "block w-full rounded-md px-3 py-2 text-left text-sm transition-colors",
                  i18n.language === 'ru' ? "bg-color-primary/10 text-color-primary font-medium" : "hover:bg-bg-hover text-text-secondary hover:text-text-primary"
                )}
              >
                Русский
              </button>
              <button
                onClick={() => changeLanguage('zh')}
                className={cn(
                  "block w-full rounded-md px-3 py-2 text-left text-sm transition-colors",
                  i18n.language === 'zh' ? "bg-color-primary/10 text-color-primary font-medium" : "hover:bg-bg-hover text-text-secondary hover:text-text-primary"
                )}
              >
                中文
              </button>
            </div>
          </div>

          {/* User Menu */}
          {user && (
            <div className="group relative ml-2 ext-left">
              <button className="flex items-center gap-2 rounded-full border border-border-primary bg-bg-secondary/50 p-1 pr-3 transition-colors hover:bg-bg-hover">
                <div className="flex h-7 w-7 items-center justify-center rounded-full bg-color-primary text-xs font-medium text-white shadow-sm">
                  {user.username.charAt(0).toUpperCase()}
                </div>
                <span className="hidden text-sm font-medium text-text-primary md:block">
                  {user.username}
                </span>
              </button>
              <div className="absolute right-0 mt-2 w-56 origin-top-right rounded-xl border border-border-primary bg-bg-primary shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
                <div className="flex flex-col space-y-1 px-4 py-3 border-b border-border-primary">
                  <p className="text-sm font-medium leading-none text-text-primary">{user.username}</p>
                  <p className="text-xs leading-none text-text-tertiary mt-1">
                    {user.is_super_admin ? t('auth.superAdmin') : t('auth.admin')}
                  </p>
                </div>
                <div className="p-1">
                  <button
                    onClick={handleLogout}
                    className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-color-danger transition-colors hover:bg-color-danger/10 font-medium"
                  >
                    <LogOut className="h-4 w-4" />
                    {t('auth.logout')}
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
