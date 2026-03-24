import { clsx } from 'clsx'
import { Home, Users, ArrowDownToLine, ArrowUpFromLine, Box, Shield, Activity, Settings, Globe, Network } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

interface SidebarProps {
  isOpen?: boolean
  onClose?: () => void
}

export function Sidebar({ isOpen = true, onClose }: SidebarProps) {
  const { t } = useTranslation()

  const navigation = [
    { name: t('nav.dashboard'), href: '/', icon: Home },
    { name: t('nav.users'), href: '/users', icon: Users },
    { name: t('nav.inbounds'), href: '/inbounds', icon: ArrowDownToLine },
    { name: t('nav.outbounds'), href: '/outbounds', icon: ArrowUpFromLine },
    { name: t('nav.cores'), href: '/cores', icon: Box },
    { name: t('nav.certificates'), href: '/certificates', icon: Shield },
    { name: t('nav.connections'), href: '/connections', icon: Activity },
    { name: t('nav.warp'), href: '/warp', icon: Network },
    { name: t('nav.geo'), href: '/geo', icon: Globe },
    { name: t('nav.settings'), href: '/settings', icon: Settings },
  ]

  return (
    <>
      {/* Mobile backdrop */}
      {isOpen && onClose && (
        <div
          className="fixed inset-0 bg-black/50 z-fixed lg:hidden"
          onClick={onClose}
        />
      )}

      {/* Sidebar */}
      <aside
        className={clsx(
          'fixed top-0 left-0 h-full w-64 bg-primary border-r border-primary',
          'transition-transform duration-base z-fixed',
          'lg:translate-x-0',
          isOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        {/* Logo */}
        <div className="h-16 flex items-center px-6 border-b border-primary">
          <h1 className="text-xl font-bold text-primary">{t('common.appName')}</h1>
        </div>

        {/* Navigation */}
        <nav className="p-4 space-y-1 overflow-y-auto h-[calc(100%-4rem)]">
          {navigation.map((item) => {
            const Icon = item.icon
            const isActive = window.location.pathname === item.href

            return (
              <a
                key={item.href}
                href={item.href}
                className={clsx(
                  'flex items-center gap-3 px-3 py-2 rounded-lg transition-base',
                  'text-sm font-medium',
                  isActive
                    ? 'bg-primary text-white'
                    : 'text-secondary hover:bg-hover hover:text-primary'
                )}
                onClick={onClose}
              >
                <Icon className="w-5 h-5 flex-shrink-0" />
                <span>{item.name}</span>
              </a>
            )
          })}
        </nav>
      </aside>
    </>
  )
}
