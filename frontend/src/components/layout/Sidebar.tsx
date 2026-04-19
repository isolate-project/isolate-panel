import { route } from 'preact-router'
import { Share2, Home, Users, ArrowDownToLine, ArrowUpFromLine, Box, Shield, Activity, Settings, Globe, Network, Database, Bell } from 'lucide-preact'
import { useTranslation } from 'react-i18next'
import { cn } from '../../lib/utils'

<<<<<<< Updated upstream
=======
function useCurrentPath() {
  const [pathname, setPathname] = useState(window.location.pathname)
  useEffect(() => {
    const interval = setInterval(() => {
      const currentPath = window.location.pathname
      if (currentPath !== pathname) {
        setPathname(currentPath)
      }
    }, 200)
    return () => clearInterval(interval)
  }, [pathname])
  return pathname
}

>>>>>>> Stashed changes
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
    { name: t('nav.backups'), href: '/backups', icon: Database },
    { name: t('nav.notifications'), href: '/notifications', icon: Bell },
  ]

  const isActivePath = (path: string) => {
    if (path === '/') return window.location.pathname === '/'
    return window.location.pathname.startsWith(path)
  }

  return (
    <>
      {/* Mobile backdrop */}
      {isOpen && onClose && (
        <div
          className="fixed inset-0 bg-black/60 backdrop-blur-sm z-[1040] lg:hidden transition-opacity"
          onClick={onClose}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-[1050] flex w-72 flex-col border-r border-border-primary bg-bg-primary/95 backdrop-blur-xl transition-transform duration-300 lg:translate-x-0',
          isOpen ? 'translate-x-0 shadow-2xl' : '-translate-x-full'
        )}
      >
        {/* Logo Area */}
        <div className="flex h-16 shrink-0 items-center px-6 border-b border-border-primary">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-color-primary bg-gradient-to-br from-indigo-500 to-indigo-600 shadow-sm">
              <Share2 className="h-5 w-5 text-white" />
            </div>
            <span className="text-lg font-semibold tracking-tight text-text-primary">
              Isolate Panel
            </span>
          </div>
        </div>

        {/* Navigation */}
        <div className="flex-1 overflow-y-auto px-4 py-6 scrollbar-none hover:scrollbar-thin scrollbar-thumb-border-secondary">
          <nav className="flex flex-col gap-1.5">
            {navigation.map((item) => {
              const Icon = item.icon
              const active = isActivePath(item.href)

              return (
                <a
                  key={item.href}
                  href={item.href}
                  onClick={(e) => { e.preventDefault(); onClose?.(); route(item.href) }}
                  className={cn(
                    'group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-200',
                    active
                      ? 'bg-color-primary/10 text-color-primary dark:bg-color-primary/15'
                      : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'
                  )}
                >
                  <Icon 
                    className={cn(
                      'h-5 w-5 shrink-0 transition-colors', 
                      active ? 'text-color-primary' : 'text-text-tertiary group-hover:text-text-primary'
                    )} 
                  />
                  {item.name}
                </a>
              )
            })}
          </nav>
        </div>

        {/* Bottom Actions */}
        <div className="mt-auto p-4 border-t border-border-primary bg-bg-secondary/50">
          <a
            href="/settings"
            onClick={(e) => { e.preventDefault(); onClose?.(); route('/settings') }}
            className={cn(
              'group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all',
              isActivePath('/settings')
                ? 'bg-color-primary/10 text-color-primary'
                : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'
            )}
          >
            <Settings className="h-5 w-5 shrink-0 text-text-tertiary group-hover:text-text-primary" />
            {t('nav.settings')}
          </a>
        </div>
      </aside>
    </>
  )
}
