import { useState, useEffect, useRef } from 'preact/hooks'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { Modal } from '../components/ui/Modal'
import { useCores, useStartCore, useStopCore, useRestartCore } from '../hooks/useCores'
import { coreApi } from '../api/endpoints'
import type { Core } from '../types'
import { Play, Square, RotateCw, FileText, Activity, RefreshCw } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

export function Cores() {
  const { t } = useTranslation()
  const { data: cores, isLoading, refetch } = useCores()
  const { mutate: startCore, isLoading: isStarting } = useStartCore()
  const { mutate: stopCore, isLoading: isStopping } = useStopCore()
  const { mutate: restartCore, isLoading: isRestarting } = useRestartCore()

  const [selectedCore, setSelectedCore] = useState<Core | null>(null)
  const [isLogsModalOpen, setIsLogsModalOpen] = useState(false)

  const handleStart = async (coreName: string) => {
    await startCore(coreName)
    refetch()
  }

  const handleStop = async (coreName: string) => {
    if (confirm(t('cores.stopConfirm'))) {
      await stopCore(coreName)
      refetch()
    }
  }

  const handleRestart = async (coreName: string) => {
    if (confirm(t('cores.restartConfirm'))) {
      await restartCore(coreName)
      refetch()
    }
  }

  const handleViewLogs = (core: Core) => {
    setSelectedCore(core)
    setIsLogsModalOpen(true)
  }

  const formatUptime = (seconds: number) => {
    if (seconds === 0) return '-'
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (hours > 0) {
      return `${hours}h ${minutes}m`
    }
    return `${minutes}m`
  }

  return (
    <PageLayout>
      <PageHeader
        title={t('cores.title')}
        description={t('cores.description')}
      />

      {isLoading ? (
        <Card className="flex items-center justify-center py-12">
      <CardContent className="p-6">
          <Spinner size="lg" />
              </CardContent>
    </Card>
      ) : cores && cores.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {cores.map((core: Core) => (
            <Card key={core.id} className="p-6">
      <CardContent className="p-6">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-semibold text-primary mb-1">
                    {core.name}
                  </h3>
                  <p className="text-sm text-tertiary">{core.type}</p>
                </div>
                <Badge variant={core.is_running ? 'success' : 'default'}>
                  {core.is_running ? t('cores.running') : t('cores.stopped')}
                </Badge>
              </div>

              <div className="space-y-2 mb-4">
                <div className="flex justify-between text-sm">
                  <span className="text-secondary">{t('cores.version')}:</span>
                  <span className="text-primary font-medium">{core.version}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-secondary">{t('cores.uptime')}:</span>
                  <span className="text-primary font-medium">
                    {formatUptime(core.uptime_seconds || 0)}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-secondary">PID:</span>
                  <span className="text-primary font-medium">
                    {core.pid || '-'}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-secondary">{t('cores.restarts')}:</span>
                  <span className="text-primary font-medium">
                    {core.restart_count || 0}
                  </span>
                </div>
              </div>

              {core.last_error && (
                <div className="mb-4 p-2 bg-danger/10 border border-danger/20 rounded text-xs text-danger">
                  {core.last_error}
                </div>
              )}

              <div className="flex gap-2">
                {!core.is_running ? (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => handleStart(core.name)}
                    disabled={isStarting}
                    className="flex-1"
                  >
                    <Play className="w-4 h-4 mr-1" />
                    {t('cores.start')}
                  </Button>
                ) : (
                  <>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => handleStop(core.name)}
                      disabled={isStopping}
                      className="flex-1"
                    >
                      <Square className="w-4 h-4 mr-1" />
                      {t('cores.stop')}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleRestart(core.name)}
                      disabled={isRestarting}
                      className="flex-1"
                    >
                      <RotateCw className="w-4 h-4 mr-1" />
                      {t('cores.restart')}
                    </Button>
                  </>
                )}
                <div title={t('cores.viewLogs')}>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleViewLogs(core)}
                  >
                    <FileText className="w-4 h-4" />
                  </Button>
                </div>
              </div>
                  </CardContent>
    </Card>
          ))}
        </div>
      ) : (
        <Card className="text-center py-12">
      <CardContent className="p-6">
          <Activity className="w-12 h-12 mx-auto mb-4 text-tertiary" />
          <p className="text-secondary">{t('cores.noCores')}</p>
              </CardContent>
    </Card>
      )}

      {/* Logs Modal */}
      <Modal
        isOpen={isLogsModalOpen}
        onClose={() => {
          setIsLogsModalOpen(false)
          setSelectedCore(null)
        }}
        title={`${selectedCore?.name} ${t('cores.logs')}`}
        size="lg"
      >
        {selectedCore && <CoreLogsView coreName={selectedCore.name} />}
      </Modal>
    </PageLayout>
  )
}

// Component to display core logs — fetches from API with polling
function CoreLogsView({ coreName }: { coreName: string }) {
  const { t } = useTranslation()
  const [logs, setLogs] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)

  const mountedRef = useRef(true)

  const fetchLogs = async () => {
    try {
      const response = await coreApi.logs(coreName, { lines: 100 })
      if (!mountedRef.current) return
      const data = response.data
      if (Array.isArray(data?.logs)) {
        setLogs(data.logs)
      } else if (typeof data?.logs === 'string') {
        setLogs(data.logs.split('\n').filter(Boolean))
      } else if (Array.isArray(data)) {
        setLogs(data)
      } else {
        setLogs([])
      }
      setError(null)
    } catch {
      if (!mountedRef.current) return
      setError(t('cores.logsError'))
      setLogs([])
    } finally {
      if (mountedRef.current) {
        setIsLoading(false)
      }
    }
  }

  useEffect(() => {
    mountedRef.current = true
    fetchLogs()
    const interval = setInterval(fetchLogs, 5000)
    return () => {
      mountedRef.current = false
      clearInterval(interval)
    }
  }, [coreName])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <p className="text-sm text-secondary">
          {t('cores.logsAutoRefresh')}
        </p>
        <Button variant="ghost" size="sm" onClick={fetchLogs}>
          <RefreshCw className="w-4 h-4 mr-1" />
          {t('common.refresh')}
        </Button>
      </div>

      {error && (
        <div className="mb-3 p-3 bg-danger/10 border border-danger/20 rounded text-sm text-danger">
          {error}
        </div>
      )}

      <div className="bg-secondary rounded-lg p-4 font-mono text-xs max-h-96 overflow-y-auto">
        {logs.length > 0 ? (
          <div className="space-y-0.5">
            {logs.map((line, i) => (
              <div
                key={i}
                className={`${
                  line.includes('[ERROR]') || line.includes('error')
                    ? 'text-danger'
                    : line.includes('[WARN]') || line.includes('warn')
                    ? 'text-warning'
                    : line.includes('[INFO]') || line.includes('info')
                    ? 'text-success'
                    : 'text-secondary'
                }`}
              >
                {line}
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        ) : (
          <div className="text-tertiary text-center py-4">
            {t('cores.noLogs')}
          </div>
        )}
      </div>
    </div>
  )
}
