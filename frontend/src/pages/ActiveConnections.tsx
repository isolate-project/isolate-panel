import { useState } from 'preact/hooks'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { Input } from '../components/ui/Input'
import { useQuery } from '../hooks/useQuery'
import { useMutation } from '../hooks/useMutation'
import { apiClient } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'
import { WifiOff, RefreshCw, LogOut } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

interface Connection {
  id: number
  user_id: number
  inbound_id: number
  core_id: number
  core_name: string
  source_ip: string
  source_port: number
  destination_ip: string
  destination_port: number
  started_at: string
  last_activity: string
  upload: number
  download: number
}

export function ActiveConnections() {
  const { t } = useTranslation()
  const { addToast } = useToastStore()
  const [userIdFilter, setUserIdFilter] = useState('')
  const [autoRefresh, setAutoRefresh] = useState(true)

  const { data, isLoading, refetch } = useQuery<{ connections: Connection[]; total: number }>(
    `active-connections-${userIdFilter}`,
    () => {
      const params = userIdFilter ? { user_id: userIdFilter } : {}
      return apiClient.get('/stats/connections', { params }).then((res) => res.data)
    },
    {
      refetchInterval: autoRefresh ? 5000 : undefined,
    }
  )

  const disconnectMutation = useMutation(
    (userId: number) => apiClient.post(`/stats/user/${userId}/disconnect`).then((res) => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: t('connections.disconnected') })
        refetch()
      },
      onError: (error: Error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatTime = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleTimeString()
  }

  const formatDuration = (startedAt: string) => {
    const start = new Date(startedAt)
    const now = new Date()
    const diff = Math.floor((now.getTime() - start.getTime()) / 1000) // seconds
    
    const hours = Math.floor(diff / 3600)
    const minutes = Math.floor((diff % 3600) / 60)
    const seconds = diff % 60
    
    if (hours > 0) return `${hours}h ${minutes}m`
    if (minutes > 0) return `${minutes}m ${seconds}s`
    return `${seconds}s`
  }

  const handleDisconnect = (userId: number) => {
    if (confirm(t('connections.confirmDisconnect'))) {
      disconnectMutation.mutate(userId)
    }
  }

  return (
    <PageLayout>
      <PageHeader
        title={t('nav.connections')}
        description={t('connections.description')}
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant={autoRefresh ? 'default' : 'secondary'}
              size="sm"
              onClick={() => setAutoRefresh(!autoRefresh)}
            >
              <RefreshCw className={`w-4 h-4 mr-1 ${autoRefresh ? 'animate-spin' : ''}`} />
              {autoRefresh ? t('connections.autoRefreshOn') : t('connections.autoRefreshOff')}
            </Button>
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCw className="w-4 h-4 mr-1" />
              {t('common.refresh')}
            </Button>
          </div>
        }
      />

      <Card className="mb-6">
      <CardContent className="p-6">
        <div className="flex items-center gap-4 mb-4">
          <div className="flex-1">
            <Input
              value={userIdFilter}
              onChange={(e) => setUserIdFilter((e.target as HTMLInputElement).value)}
              placeholder={t('connections.filterByUserId')}
            />
          </div>
          <div className="text-sm text-secondary">
            {data?.total || 0} {t('connections.activeConnections')}
          </div>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-12">
            <Spinner size="lg" />
          </div>
        ) : data?.connections.length === 0 ? (
          <div className="text-center py-12">
            <WifiOff className="w-16 h-16 mx-auto text-secondary mb-4" />
            <h3 className="text-lg font-medium text-primary mb-2">{t('connections.noConnections')}</h3>
            <p className="text-secondary">{t('connections.noConnectionsDesc')}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-primary">
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('connections.user')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('connections.core')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('connections.source')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('connections.destination')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('connections.duration')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('connections.traffic')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {data?.connections.map((conn) => (
                  <tr key={conn.id} className="border-b border-hover hover:bg-hover/50">
                    <td className="py-3 px-4">
                      <span className="font-medium text-primary">User #{conn.user_id}</span>
                    </td>
                    <td className="py-3 px-4">
                      <Badge variant="outline">{conn.core_name}</Badge>
                    </td>
                    <td className="py-3 px-4">
                      <div className="text-sm text-primary">{conn.source_ip}</div>
                      <div className="text-xs text-tertiary">:{conn.source_port}</div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="text-sm text-primary">{conn.destination_ip}</div>
                      <div className="text-xs text-tertiary">:{conn.destination_port}</div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="text-sm text-primary">{formatDuration(conn.started_at)}</div>
                      <div className="text-xs text-tertiary">{formatTime(conn.last_activity)}</div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="text-xs">
                        <span className="text-green-500">↑ {formatBytes(conn.upload)}</span>
                        <br />
                        <span className="text-blue-500">↓ {formatBytes(conn.download)}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDisconnect(conn.user_id)}
                        disabled={disconnectMutation.isLoading}
                      >
                        <LogOut className="w-3 h-3 mr-1" />
                        {t('connections.disconnect')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
            </CardContent>
    </Card>
    </PageLayout>
  )
}
