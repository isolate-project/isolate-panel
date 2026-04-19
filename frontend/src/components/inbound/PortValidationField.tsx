import { useState, useCallback, useRef, useEffect } from 'preact/hooks'
import { useCheckPort, type PortConflict, type PortValidationResult } from '../../hooks/useCheckPort'
import { useTranslation } from 'react-i18next'

interface PortValidationFieldProps {
  value: number
  onChange: (port: number) => void
  protocol: string
  transport?: string
  coreType: string
  listenAddress?: string
  disabled?: boolean
}

type ValidationStatus = 'idle' | 'checking' | 'success' | 'warning' | 'error'

interface ValidationState {
  status: ValidationStatus
  message: string
  action: 'allow' | 'confirm' | 'block'
  canShare: boolean
  conflicts?: PortConflict[]
}

export function PortValidationField({
  value,
  onChange,
  protocol,
  transport = '',
  coreType,
  listenAddress = '0.0.0.0',
  disabled = false,
}: PortValidationFieldProps) {
  const { t } = useTranslation()
  const { mutate: checkPort, isLoading, data, error, reset } = useCheckPort()
  const [state, setState] = useState<ValidationState>({
    status: 'idle',
    message: '',
    action: 'allow',
    canShare: false,
  })
  const debounceRef = useRef<number | null>(null)
  const pendingPortRef = useRef<number | null>(null)

  useEffect(() => {
    if (data && pendingPortRef.current !== null) {
      const result = data as PortValidationResult
      setState({
        status:
          result.severity === 'error'
            ? 'error'
            : result.severity === 'warning'
              ? 'warning'
              : 'success',
        message: result.message,
        action: result.action,
        canShare: result.can_share_port,
        conflicts: result.conflicts,
      })
      pendingPortRef.current = null
    }
  }, [data])

  useEffect(() => {
    if (error) {
      setState({
        status: 'error',
        message: t('inbounds.portCheckError') || 'Failed to check port',
        action: 'block',
        canShare: false,
      })
      pendingPortRef.current = null
    }
  }, [error, t])

  const validatePort = useCallback(
    (port: number) => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }

      if (!port || port < 1 || port > 65535) {
        setState({
          status: 'error',
          message: t('inbounds.portOutOfRange') || 'Port must be between 1 and 65535',
          action: 'block',
          canShare: false,
        })
        return
      }

      setState((prev) => ({ ...prev, status: 'checking' }))
      reset()
      pendingPortRef.current = port

      debounceRef.current = window.setTimeout(() => {
        checkPort({
          port,
          listen: listenAddress,
          protocol,
          transport,
          core_type: coreType,
        })
      }, 500)
    },
    [checkPort, protocol, transport, coreType, listenAddress, t, reset]
  )

  useEffect(() => {
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (value && value > 0) {
      validatePort(value)
    }
  }, [value, protocol, transport, coreType, listenAddress, validatePort])

  const handleChange = (e: Event) => {
    const target = e.target as HTMLInputElement
    const newPort = parseInt(target.value, 10)
    onChange(isNaN(newPort) ? 0 : newPort)
  }

  const getStatusColor = () => {
    switch (state.status) {
      case 'success':
        return 'text-green-500 border-green-500'
      case 'warning':
        return 'text-yellow-500 border-yellow-500'
      case 'error':
        return 'text-red-500 border-red-500'
      case 'checking':
        return 'text-blue-500 border-blue-500'
      default:
        return ''
    }
  }

  const getIcon = () => {
    switch (state.status) {
      case 'success':
        return '✓'
      case 'warning':
        return '⚠️'
      case 'error':
        return '✗'
      case 'checking':
        return '⏳'
      default:
        return ''
    }
  }

  const isBlocked = state.action === 'block'

  return (
    <div className="space-y-2">
      <div className="relative">
        <input
          type="number"
          value={value || ''}
          onInput={handleChange}
          min={1}
          max={65535}
          disabled={disabled || isLoading}
          className={`w-full rounded-md border px-3 py-2 text-sm ${getStatusColor()} ${disabled || isLoading ? 'opacity-50' : ''}`}
          placeholder={t('inbounds.portPlaceholder') || '443'}
        />
        {state.status === 'checking' && (
          <span className="absolute right-3 top-2 animate-spin">⏳</span>
        )}
      </div>

      {state.message && (
        <div
          className={`text-sm ${getStatusColor()} flex items-start gap-2`}
        >
          <span className="mt-0.5">{getIcon()}</span>
          <div className="flex-1">
            <p>{state.message}</p>

            {state.conflicts && state.conflicts.length > 0 && (
              <div className="mt-2 text-xs text-gray-600">
                <p className="font-medium mb-1">
                  {t('inbounds.existingInbounds') || 'Existing inbounds on this port:'}
                </p>
                <ul className="space-y-1">
                  {state.conflicts.map((conflict) => (
                    <li key={conflict.inbound_id} className="flex items-center gap-2">
                      <span>
                        {conflict.inbound_name} ({conflict.protocol}
                        {conflict.transport ? `/${conflict.transport}` : ''})
                      </span>
                      {conflict.can_share ? (
                        <span className="text-green-600 text-xs bg-green-100 px-1.5 py-0.5 rounded">
                          ✓ HAProxy OK
                        </span>
                      ) : (
                        <span className="text-red-600 text-xs bg-red-100 px-1.5 py-0.5 rounded">
                          ✗ Incompatible
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        </div>
      )}

      {state.status === 'warning' && state.action === 'confirm' && (
        <div className="flex gap-2 mt-2">
          <button
            type="button"
            onClick={() => onChange(0)}
            className="px-3 py-1.5 text-sm border rounded hover:bg-gray-50"
          >
            {t('inbounds.chooseDifferentPort') || 'Choose different port'}
          </button>
          <button
            type="button"
            className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            {t('inbounds.proceedWithHaproxy') || 'Proceed with HAProxy'}
          </button>
        </div>
      )}

      {isBlocked && (
        <div className="text-xs text-gray-500 mt-1">
          {t('inbounds.udpPortsHint') ||
            'UDP/QUIC protocols (Hysteria2, TUIC) must use separate ports. Use port 8444+ for these inbounds.'}
        </div>
      )}
    </div>
  )
}
