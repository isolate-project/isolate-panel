import { useState } from 'preact/hooks'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { Alert } from '../ui/Alert'
import { Spinner } from '../ui/Spinner'
import { AlertTriangle, Zap } from 'lucide-preact'
import { useSystemResources } from '../../hooks/useSystem'
import { useMutation } from '../../hooks/useMutation'
import { systemApi } from '../../api/endpoints'
import { sanitizeError } from '../../utils/errorHandler'
import { useToastStore } from '../../stores/toastStore'
import { useTranslation } from 'react-i18next'

interface CleanupResult {
  freed_mb?: number
}

export function RAMPanicButton() {
  const { t } = useTranslation()
  const { data: resources } = useSystemResources()
  const { addToast } = useToastStore()
  const [result, setResult] = useState<string | null>(null)

  const { mutate: emergencyCleanup, isLoading } = useMutation(
    () => systemApi.emergencyCleanup().then((res) => res.data),
    {
      onSuccess: (data: CleanupResult) => {
        const freedMB = data.freed_mb || 0
        setResult(`${t('common.success')}: ${freedMB}MB`)
        addToast({
          type: 'success',
          message: `${t('ramPanic.cleanupSuccess', { mb: freedMB })}`,
        })
      },
      onError: (error) => {
        const { message } = sanitizeError(error)
        setResult(`${t('common.error')}: ${message}`)
        addToast({
          type: 'error',
          message: `${t('ramPanic.cleanupFailed')}: ${message}`,
        })
      },
    }
  )

  const handlePanic = () => {
    if (!confirm(t('ramPanic.confirmMessage'))) {
      return
    }

    setResult(null)
    emergencyCleanup({})
  }

  if (!resources) {
    return (
      <Card className="border-2 border-red-500">
        <div className="flex items-center justify-center py-8">
          <Spinner size="md" />
        </div>
      </Card>
    )
  }

  const ramPercent = resources.ram?.percent || 0
  const isWarning = ramPercent > 70
  const isCritical = ramPercent > 85

  return (
    <Card className="border-2 border-red-500">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold flex items-center gap-2">
          <AlertTriangle className="text-danger" />
          {t('ramPanic.title')}
        </h3>
      </div>

      {/* RAM Usage */}
      <div className="mb-4">
        <div className="flex justify-between text-sm mb-1">
          <span>{t('ramPanic.ramUsage')}</span>
          <span
            className={
              isCritical
                ? 'text-danger font-bold'
                : isWarning
                ? 'text-warning font-semibold'
                : 'text-success'
            }
          >
            {resources.ram?.used || 0}MB / {resources.ram?.total || 0}MB (
            {ramPercent}%)
          </span>
        </div>
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
          <div
            className={`h-3 rounded-full transition-all ${
              isCritical
                ? 'bg-danger'
                : isWarning
                ? 'bg-warning'
                : 'bg-success'
            }`}
            style={{ width: `${ramPercent}%` }}
          />
        </div>
      </div>

      {/* CPU Usage */}
      <div className="mb-4">
        <div className="flex justify-between text-sm mb-1">
          <span>{t('ramPanic.cpuUsage')}</span>
          <span>{resources.cpu?.percent || 0}%</span>
        </div>
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
          <div
            className="bg-primary h-2 rounded-full transition-all"
            style={{ width: `${resources.cpu?.percent || 0}%` }}
          />
        </div>
      </div>

      {(isWarning || isCritical) && (
        <Alert variant={isCritical ? 'danger' : 'warning'} className="mb-4">
          {isCritical ? (
            <>
              <strong>{t('ramPanic.critical')}:</strong> {t('ramPanic.criticalMessage')}
            </>
          ) : (
            <>
              <strong>{t('ramPanic.warning')}:</strong> {t('ramPanic.warningMessage')}
            </>
          )}
        </Alert>
      )}

      <Button
        variant="destructive"
        onClick={handlePanic}
        disabled={isLoading}
        fullWidth
      >
        {isLoading ? (
          <>
            <Spinner className="mr-2" size="sm" />
            {t('ramPanic.cleaning')}
          </>
        ) : (
          <>
            <Zap className="mr-2" />
            {t('ramPanic.button')}
          </>
        )}
      </Button>

      {result && (
        <div
          className={`mt-3 text-sm ${
            result.startsWith(t('common.success')) ? 'text-success' : 'text-danger'
          }`}
        >
          {result}
        </div>
      )}

      <div className="mt-3 text-xs text-tertiary">
        <strong>{t('ramPanic.whatThisDoes')}:</strong>
        <ul className="list-disc list-inside mt-1">
          <li>{t('ramPanic.clearsSubCache')}</li>
          <li>{t('ramPanic.clearsConfigCache')}</li>
          <li>{t('ramPanic.restartsCores')}</li>
          <li>{t('ramPanic.forcesGC')}</li>
        </ul>
      </div>
    </Card>
  )
}
