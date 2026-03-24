import { route } from 'preact-router'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { RAMPanicButton } from '../components/features/RAMPanicButton'
import { useUsers } from '../hooks/useUsers'
import { useCores } from '../hooks/useCores'
import { useSystemResources } from '../hooks/useSystem'
import { useConnections } from '../hooks/useConnections'
import type { User, Core } from '../types'
import { Users, Activity, HardDrive, Box } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

export function Dashboard() {
  const { t } = useTranslation()
  const { data: usersResponse, isLoading: usersLoading } = useUsers()
  const { data: cores, isLoading: coresLoading } = useCores()
  const { data: resources } = useSystemResources()
  const { count: activeConnections, isLoading: connectionsLoading } = useConnections()

  const users = usersResponse?.users || usersResponse || []
  const totalUsers = Array.isArray(users) ? users.length : 0
  const activeUsers = Array.isArray(users) ? users.filter((u: User) => u.is_active)?.length : 0
  const runningCores = Array.isArray(cores) ? cores.filter((c: Core) => c.is_running)?.length : 0
  const totalCores = Array.isArray(cores) ? cores.length : 0

  // Calculate total traffic from all users
  const totalTrafficBytes = Array.isArray(users)
    ? users.reduce((sum: number, u: User) => sum + (u.traffic_used_bytes || 0), 0)
    : 0
  const formatTraffic = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  return (
    <PageLayout>
      <PageHeader
        title={t('dashboard.title')}
        description={t('dashboard.overview')}
      />

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
        {/* Total Users */}
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-secondary mb-1">
                {t('dashboard.totalUsers')}
              </p>
              {usersLoading ? (
                <Spinner size="sm" />
              ) : (
                <p className="text-2xl font-bold text-primary">{totalUsers}</p>
              )}
              <p className="text-xs text-tertiary mt-1">
                {activeUsers} {t('common.active').toLowerCase()}
              </p>
            </div>
            <div className="p-3 bg-blue-100 dark:bg-blue-900 rounded-lg">
              <Users className="w-6 h-6 text-primary" />
            </div>
          </div>
        </Card>

        {/* Active Connections */}
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-secondary mb-1">
                {t('dashboard.activeConnections')}
              </p>
              {connectionsLoading ? (
                <Spinner size="sm" />
              ) : (
                <p className="text-2xl font-bold text-primary">{activeConnections}</p>
              )}
              <p className="text-xs text-tertiary mt-1">{t('dashboard.realtime')}</p>
            </div>
            <div className="p-3 bg-green-100 dark:bg-green-900 rounded-lg">
              <Activity className="w-6 h-6 text-success" />
            </div>
          </div>
        </Card>

        {/* Total Traffic */}
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-secondary mb-1">
                {t('dashboard.totalTraffic')}
              </p>
              {usersLoading ? (
                <Spinner size="sm" />
              ) : (
                <p className="text-2xl font-bold text-primary">{formatTraffic(totalTrafficBytes)}</p>
              )}
              <p className="text-xs text-tertiary mt-1">{t('dashboard.allUsers')}</p>
            </div>
            <div className="p-3 bg-purple-100 dark:bg-purple-900 rounded-lg">
              <HardDrive className="w-6 h-6 text-purple-600 dark:text-purple-400" />
            </div>
          </div>
        </Card>

        {/* Cores Running */}
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-secondary mb-1">
                {t('dashboard.coreStatus')}
              </p>
              {coresLoading ? (
                <Spinner size="sm" />
              ) : (
                <p className="text-2xl font-bold text-primary">
                  {runningCores}/{totalCores}
                </p>
              )}
              <p className="text-xs text-tertiary mt-1">{t('dashboard.coresRunning')}</p>
            </div>
            <div className="p-3 bg-yellow-100 dark:bg-yellow-900 rounded-lg">
              <Box className="w-6 h-6 text-warning" />
            </div>
          </div>
        </Card>
      </div>

      {/* System Resources + Quick Actions */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* System Resources */}
        <Card>
          <h3 className="text-lg font-semibold text-primary mb-4">
            {t('dashboard.systemResources')}
          </h3>
          {resources ? (
            <div className="space-y-4">
              {/* RAM Usage */}
              <div>
                <div className="flex justify-between text-sm mb-2">
                  <span className="text-secondary">{t('dashboard.ramUsage')}</span>
                  <span className="font-medium text-primary">
                    {resources.ram?.used || 0}MB / {resources.ram?.total || 0}MB
                    ({resources.ram?.percent || 0}%)
                  </span>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                  <div
                    className={`h-3 rounded-full transition-all ${
                      (resources.ram?.percent || 0) > 85
                        ? 'bg-danger'
                        : (resources.ram?.percent || 0) > 70
                        ? 'bg-warning'
                        : 'bg-success'
                    }`}
                    style={{ width: `${resources.ram?.percent || 0}%` }}
                  />
                </div>
              </div>

              {/* CPU Usage */}
              <div>
                <div className="flex justify-between text-sm mb-2">
                  <span className="text-secondary">{t('dashboard.cpuUsage')}</span>
                  <span className="font-medium text-primary">
                    {resources.cpu?.percent || 0}%
                  </span>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                  <div
                    className="bg-primary h-3 rounded-full transition-all"
                    style={{ width: `${resources.cpu?.percent || 0}%` }}
                  />
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center py-4">
              <Spinner size="md" />
            </div>
          )}
        </Card>

        {/* Quick Actions */}
        <Card>
          <h3 className="text-lg font-semibold text-primary mb-4">
            {t('dashboard.quickActions')}
          </h3>
          <div className="space-y-3">
            <Button
              variant="primary"
              fullWidth
              onClick={() => route('/users')}
            >
              <Users className="w-4 h-4 mr-2" />
              {t('users.addUser')}
            </Button>
            <Button
              variant="secondary"
              fullWidth
              onClick={() => route('/inbounds')}
            >
              {t('inbounds.addInbound')}
            </Button>
            <Button
              variant="secondary"
              fullWidth
              onClick={() => route('/cores')}
            >
              {t('dashboard.manageCores')}
            </Button>
          </div>
        </Card>
      </div>

      {/* RAM Panic Button */}
      <div className="mb-6">
        <RAMPanicButton />
      </div>

      {/* Core Status Cards */}
      {cores && Array.isArray(cores) && cores.length > 0 && (
        <Card>
          <h3 className="text-lg font-semibold text-primary mb-4">
            {t('dashboard.proxyCores')}
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {cores.map((core: Core) => (
              <div
                key={core.id}
                className="p-4 border border-primary rounded-lg"
              >
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-semibold text-primary">{core.name}</h4>
                  <Badge
                    variant={core.is_running ? 'success' : 'default'}
                  >
                    {core.is_running ? t('cores.running') : t('cores.stopped')}
                  </Badge>
                </div>
                <p className="text-sm text-secondary">
                  {t('cores.version')}: {core.version || 'N/A'}
                </p>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Welcome Message for Empty State */}
      {!usersLoading && totalUsers === 0 && (
        <Card className="mt-6">
          <h2 className="text-lg font-semibold text-primary mb-4">
            {t('dashboard.quickActions')}
          </h2>
          <p className="text-secondary mb-4">
            {t('dashboard.welcomeMessage')}
          </p>
          <Button
            variant="primary"
            onClick={() => route('/users')}
          >
            {t('dashboard.createFirstUser')}
          </Button>
        </Card>
      )}
    </PageLayout>
  )
}
