import type { ComponentChildren } from 'preact'
import { route } from 'preact-router'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Progress } from '../components/ui/Progress'
import { Skeleton } from '../components/ui/Skeleton'
import { RAMPanicButton } from '../components/features/RAMPanicButton'
import { TrafficChart, TopUsersChart } from '../components/features/DashboardCharts'
import { useUsers } from '../hooks/useUsers'
import { useCores } from '../hooks/useCores'
import { useSystemResources } from '../hooks/useSystem'
import { useConnections } from '../hooks/useConnections'
import type { User, Core } from '../types'
import { Users, Activity, HardDrive, Box, ArrowUpRight, ShieldAlert, Cpu, LucideIcon } from 'lucide-preact'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useMetaTags } from '../hooks/useDocumentTitle'


export function Dashboard() {
  const { t } = useTranslation()

  useMetaTags({
    title: t('dashboard.title') || 'Dashboard',
    description: 'Isolate Panel dashboard — proxy servers overview, active connections, and system resources monitoring',
  })

  const { data: usersResponse, isLoading: usersLoading } = useUsers()
  const { data: cores, isLoading: coresLoading } = useCores()
  const { data: resources } = useSystemResources()
  const { count: activeConnections, isLoading: connectionsLoading } = useConnections()

  const users = usersResponse?.users || usersResponse || []
  const totalUsers = Array.isArray(users) ? users.length : 0
  const activeUsers = Array.isArray(users) ? users.filter((u: User) => u.is_active)?.length : 0
  const runningCores = Array.isArray(cores) ? cores.filter((c: Core) => c.is_running)?.length : 0
  const totalCores = Array.isArray(cores) ? cores.length : 0

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


interface StatCardProps {
  title: string
  value: string | number
  subtext?: ComponentChildren
  icon: LucideIcon
  loading?: boolean
  colorClass?: string
}

const StatCard = ({ title, value, subtext, icon: Icon, loading, colorClass }: StatCardProps) => (
  <Card className="hover:shadow-lg transition-all duration-200">
    <CardContent className="p-6">
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <p className="text-sm font-medium text-text-secondary">{title}</p>
          {loading ? (
            <Skeleton className="h-8 w-24" />
          ) : (
            <p className="text-3xl font-bold tracking-tight text-text-primary">{value}</p>
          )}
          <p className="text-xs font-medium text-text-tertiary flex items-center gap-1">
            {subtext}
          </p>
        </div>
        <div className={cn("p-4 rounded-2xl flex-shrink-0", colorClass)}>
          <Icon className="w-6 h-6" />
        </div>
      </div>
    </CardContent>
  </Card>
)

  const ramPercent = resources?.ram?.percent || 0
  const cpuPercent = resources?.cpu?.percent || 0

  return (
    <PageLayout>
      <div className="flex flex-col gap-6">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <PageHeader
            title={t('dashboard.title')}
            description={t('dashboard.overview')}
          />
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => route('/inbounds/create')}>
              {t('inbounds.addInbound')}
            </Button>
            <Button onClick={() => route('/users')}>
              <Users className="w-4 h-4 mr-2" />
              {t('users.addUser')}
            </Button>
          </div>
        </div>

        {/* 1. Main Stats Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 lg:gap-6">
          <StatCard
            title={t('dashboard.totalUsers')}
            value={totalUsers}
            subtext={
              <><span className="text-color-success">{activeUsers}</span> active users</>
            }
            icon={Users}
            loading={usersLoading}
            colorClass="bg-indigo-50 text-indigo-600 dark:bg-indigo-500/10 dark:text-indigo-400"
          />
          <StatCard
            title={t('dashboard.activeConnections')}
            value={activeConnections}
            subtext={<><Activity className="w-3 h-3" /> Realtime tracking</>}
            icon={Activity}
            loading={connectionsLoading}
            colorClass="bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400"
          />
          <StatCard
            title={t('dashboard.totalTraffic')}
            value={formatTraffic(totalTrafficBytes)}
            subtext="Up and down combined"
            icon={ArrowUpRight}
            loading={usersLoading}
            colorClass="bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400"
          />
          <StatCard
            title={t('dashboard.coreStatus')}
            value={`${runningCores}/${totalCores}`}
            subtext="Engines running"
            icon={Box}
            loading={coresLoading}
            colorClass="bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400"
          />
        </div>

        {/* 2. Traffic Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <TrafficChart days={7} />
          </div>
          <TopUsersChart limit={5} />
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main System Resources - Spans 2 cols */}
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <HardDrive className="w-5 h-5 text-text-secondary" />
                {t('dashboard.systemResources')}
              </CardTitle>
              <CardDescription>Server hardware utilization</CardDescription>
            </CardHeader>
            <CardContent>
              {resources ? (
                <div className="grid sm:grid-cols-2 gap-8">
                  {/* CPU Gauge (Linear fallback for now) */}
                  <div className="space-y-4">
                    <div className="flex justify-between items-end">
                      <div className="space-y-1">
                        <p className="text-sm font-medium text-text-secondary">Processor Usage</p>
                        <p className="text-2xl font-bold tracking-tight text-text-primary">{cpuPercent}%</p>
                      </div>
                      <Cpu className="w-8 h-8 text-text-tertiary opacity-50" />
                    </div>
                    <Progress 
                      value={cpuPercent} 
                      className="h-3"
                      indicatorClassName={
                        cpuPercent > 80 ? 'bg-color-danger' : 
                        cpuPercent > 60 ? 'bg-color-warning' : 
                        'bg-color-primary'
                      }
                    />
                  </div>

                  {/* RAM Linear */}
                  <div className="space-y-4">
                    <div className="flex justify-between items-end">
                      <div className="space-y-1">
                        <p className="text-sm font-medium text-text-secondary">Memory Status</p>
                        <p className="text-2xl font-bold tracking-tight text-text-primary">
                          {resources.ram?.used || 0} <span className="text-base font-medium text-text-tertiary">/ {resources.ram?.total || 0} MB</span>
                        </p>
                      </div>
                    </div>
                    <div className="space-y-1.5">
                      <Progress 
                        value={ramPercent} 
                        className="h-3 bg-bg-tertiary"
                        indicatorClassName={
                          ramPercent > 85 ? 'bg-color-danger' : 
                          ramPercent > 70 ? 'bg-color-warning' : 
                          'bg-color-success'
                        }
                      />
                      <div className="flex justify-between text-xs text-text-tertiary font-medium">
                        <span>Used: {resources.ram?.used || 0}MB</span>
                        <span>{ramPercent}%</span>
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col gap-8">
                  <div className="space-y-2"><Skeleton className="h-4 w-32"/><Skeleton className="h-3 w-full"/></div>
                  <div className="space-y-2"><Skeleton className="h-4 w-32"/><Skeleton className="h-3 w-full"/></div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Quick Actions & Panic - Spans 1 col */}
          <div className="flex flex-col gap-6">
            <Card className="border-color-danger/20 bg-danger/5 dark:bg-danger/10 overflow-hidden">
              <div className="absolute top-0 right-0 p-4 opacity-10">
                <ShieldAlert className="w-24 h-24 text-color-danger" />
              </div>
              <CardHeader className="pb-3 relative z-10">
                <CardTitle className="text-color-danger flex items-center gap-2">
                  <ShieldAlert className="w-5 h-5" />
                  Emergency
                </CardTitle>
                <CardDescription>Free up system memory instantly</CardDescription>
              </CardHeader>
              <CardContent className="relative z-10 pb-6 pt-2">
                <RAMPanicButton />
              </CardContent>
            </Card>

            <Card className="flex-1">
              <CardHeader className="pb-4">
                <CardTitle>{t('dashboard.proxyCores')}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {coresLoading ? (
                  Array(2).fill(0).map((_, i) => <Skeleton key={i} className="h-16 w-full rounded-xl" />)
                ) : cores && cores.length > 0 ? (
                  cores.slice(0, 3).map((core: Core) => (
                    <div key={core.id} className="flex items-center justify-between p-3 rounded-xl border border-border-primary bg-bg-primary hover:bg-bg-hover transition-colors cursor-pointer" onClick={() => route('/cores')}>
                      <div className="flex items-center gap-3">
                        <div className={cn("w-2 h-2 rounded-full", core.is_running ? "bg-color-success" : "bg-color-danger animate-pulse")} />
                        <div>
                          <p className="text-sm font-semibold text-text-primary">{core.name}</p>
                          <p className="text-xs text-text-secondary">{core.version || 'v1.0'}</p>
                        </div>
                      </div>
                      <Badge variant={core.is_running ? 'success' : 'secondary'} showDot className="text-[10px] lowercase">
                        {core.is_running ? t('cores.running') : t('cores.stopped')}
                      </Badge>
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-text-tertiary text-center py-4">No cores installed</p>
                )}
                
                {cores && cores.length > 3 && (
                  <Button variant="ghost" fullWidth className="text-xs mt-2" onClick={() => route('/cores')}>
                    View all cores...
                  </Button>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </PageLayout>
  )
}
