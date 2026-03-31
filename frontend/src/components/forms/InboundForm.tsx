import { useState, useEffect, useRef } from 'preact/hooks'
import { useForm } from '../../hooks/useForm'
import { inboundSchema, InboundFormData } from '../../utils/validators'
import { FormField } from './FormField'
import { Button } from '../ui/Button'
import { useCreateInbound, useUpdateInbound } from '../../hooks/useInbounds'
import { useCores } from '../../hooks/useCores'
import { useQuery } from '../../hooks/useQuery'
import type { Inbound, Core } from '../../types'
import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '../ui/Card'
import { Save } from 'lucide-preact'
import { inboundApi, certificateApi } from '../../api/endpoints'

interface InboundFormProps {
  inbound?: Inbound | null
  onSuccess: () => void
  onCancel: () => void
}

// Protocol support matrix per core type
const CORE_PROTOCOLS: Record<string, string[]> = {
  'sing-box': ['vless', 'vmess', 'trojan', 'shadowsocks', 'hysteria2', 'tuic', 'naive', 'http', 'socks'],
  'xray': ['vless', 'vmess', 'trojan', 'shadowsocks', 'dokodemo-door', 'socks', 'http', 'mtproto'],
  'mihomo': ['vless', 'vmess', 'trojan', 'shadowsocks', 'hysteria2', 'tuic', 'http', 'socks'],
}

export function InboundForm({ inbound, onSuccess, onCancel }: InboundFormProps) {
  const { t } = useTranslation()
  const { mutate: createInbound, isLoading: isCreating } = useCreateInbound()
  const { mutate: updateInbound, isLoading: isUpdating } = useUpdateInbound()
  const { data: cores } = useCores()

  const coreOptions = Array.isArray(cores)
    ? cores.map((c: Core) => ({ value: c.id.toString(), label: `${c.name} (${c.type})`, type: c.type }))
    : []

  const [selectedCoreType, setSelectedCoreType] = useState<string>('')
  const [portError, setPortError] = useState<string | null>(null)
  const portCheckTimeout = useRef<number | null>(null)

  // Fetch available certificates for dropdown
  const { data: certDropdownData } = useQuery<{ options: Array<{ id: number; domain: string; label: string }> }>(
    'certificates-dropdown',
    () => certificateApi.dropdown().then(res => res.data as { options: Array<{ id: number; domain: string; label: string }> })
  )
  const certOptions = (certDropdownData?.options ?? []).map(c => ({ value: c.id.toString(), label: c.label }))

  const {
    values,
    errors,
    touched,
    isSubmitting,
    handleChange,
    handleBlur,
    handleSubmit,
  } = useForm<InboundFormData>({
    schema: inboundSchema,
    initialValues: inbound
      ? {
          name: inbound.name,
          protocol: inbound.protocol as InboundFormData['protocol'],
          port: inbound.port,
          core_id: inbound.core_id,
          listen_address: inbound.listen_address || '0.0.0.0',
          is_enabled: inbound.is_enabled ?? true,
          tls_enabled: inbound.tls_enabled ?? true,
          tls_cert_id: inbound.tls_cert_id ?? null,
        }
      : {
          name: '',
          protocol: 'vless' as const,
          port: 443,
          core_id: Number(coreOptions[0]?.value || '1'),
          listen_address: '0.0.0.0',
          is_enabled: true,
          tls_enabled: true,
          tls_cert_id: null,
        },
    onSubmit: async (data) => {
      const payload = data as unknown as Record<string, unknown>
      // Convert sentinel 0 to null for API
      if (!payload.tls_cert_id) {
        payload.tls_cert_id = null
      }
      if (inbound) {
        await updateInbound({ id: inbound.id, data: payload })
      } else {
        await createInbound(payload)
      }
      onSuccess()
    },
  })

  const isLoading = Boolean(isSubmitting || isCreating || isUpdating)
  const isInvalid = !!portError || Object.keys(errors).length > 0

  const onChange = (name: string, value: string | number | boolean) => {
    handleChange(name as keyof InboundFormData, value)
    
    // Update core type when core changes
    if (name === 'core_id') {
      const selectedCore = coreOptions.find(c => c.value === String(value))
      if (selectedCore?.type) {
        setSelectedCoreType(selectedCore.type)
        // Reset protocol to first supported if current is not supported
        const supportedProtocols = CORE_PROTOCOLS[selectedCore.type] || []
        if (!supportedProtocols.includes(values.protocol as string)) {
          handleChange('protocol', (supportedProtocols[0] || 'vless') as InboundFormData['protocol'])
        }
      }
    }

    // Clear certificate when TLS is disabled
    if (name === 'tls_enabled' && value === false) {
      handleChange('tls_cert_id', 0)
    }

    // Debounced port check
    if (name === 'port') {
      const newPort = Number(value)
      setPortError(null)

      if (portCheckTimeout.current) {
        window.clearTimeout(portCheckTimeout.current)
      }

      if (newPort < 1024 || newPort > 65535) {
        setPortError(t('inbounds.portOutOfRange') || 'Port must be between 1024 and 65535')
        return
      }

      portCheckTimeout.current = window.setTimeout(async () => {
        try {
          const res = await inboundApi.checkPort(newPort, inbound?.id)
          if (!res.data.available) {
            setPortError(res.data.reason)
          }
        } catch (err) {
          console.error('Failed to check port:', err)
        }
      }, 500)
    }
  }
  
  const onBlur = (name: string) => {
    handleBlur(name as keyof InboundFormData)
  }

  // Initialize core type on mount
  useEffect(() => {
    if (inbound && cores) {
      const core = cores.find((c: Core) => c.id === inbound.core_id)
      if (core) setSelectedCoreType(core.type)
    } else if (coreOptions.length > 0) {
      setSelectedCoreType(coreOptions[0]?.type || '')
    }
  }, [inbound, cores])

  // Get supported protocols for selected core
  const supportedProtocols = CORE_PROTOCOLS[selectedCoreType] || []
  const protocolOptions = supportedProtocols.map(p => ({ value: p, label: p.toUpperCase() }))

  return (
    <form onSubmit={handleSubmit} className="flex flex-col h-full">
      <div className="flex-1 space-y-6 pb-6">
        
        {/* Core Settings */}
        <Card className="border border-border-primary bg-bg-secondary shadow-sm">
          <CardContent className="p-5 space-y-5">
            <h3 className="font-semibold text-text-primary text-sm mb-4 uppercase tracking-wider">General Configuration</h3>
            
            <FormField
              name="name"
              label={t('inbounds.name')}
              type="text"
              value={values.name}
              error={errors.name}
              touched={touched.name}
              required
              disabled={isLoading}
              placeholder="e.g. Europe-VLESS"
              onChange={onChange}
              onBlur={onBlur}
            />

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <FormField
                name="protocol"
                label={t('inbounds.protocol')}
                type="select"
                value={values.protocol}
                error={errors.protocol}
                touched={touched.protocol}
                required
                disabled={isLoading}
                options={protocolOptions.length > 0 ? protocolOptions : [{ value: 'vless', label: 'VLESS' }]}
                onChange={onChange}
                onBlur={onBlur}
              />

              <FormField
                name="core_id"
                label={t('inbounds.core')}
                type="select"
                value={String(values.core_id)}
                error={errors.core_id}
                touched={touched.core_id}
                required
                disabled={isLoading}
                options={coreOptions}
                onChange={onChange}
                onBlur={onBlur}
              />
            </div>
          </CardContent>
        </Card>

        {/* Network Settings */}
        <Card className="border border-border-primary bg-bg-secondary shadow-sm">
          <CardContent className="p-5 space-y-5">
            <h3 className="font-semibold text-text-primary text-sm mb-4 uppercase tracking-wider">Network Settings</h3>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <FormField
                name="listen_address"
                label={t('inbounds.listenAddress')}
                type="text"
                value={values.listen_address}
                error={errors.listen_address}
                touched={touched.listen_address}
                required
                disabled={isLoading}
                placeholder="0.0.0.0"
                helperText="Use 0.0.0.0 to listen on all interfaces"
                onChange={onChange}
                onBlur={onBlur}
              />

              <FormField
                name="port"
                label={t('inbounds.port')}
                type="number"
                value={values.port}
                error={errors.port}
                touched={touched.port}
                required
                disabled={isLoading}
                placeholder="443"
                className={portError ? 'border-red-500' : ''}
                onChange={onChange}
                onBlur={onBlur}
              />
              {portError && <p className="text-xs text-red-500 mt-1">{portError}</p>}
            </div>
          </CardContent>
        </Card>

        {/* Toggle Settings */}
        <Card className="border border-border-primary bg-bg-secondary shadow-sm">
          <CardContent className="p-5 space-y-4">
            <h3 className="font-semibold text-text-primary text-sm mb-4 uppercase tracking-wider">Security & Status</h3>

            <FormField
              name="tls_enabled"
              label="Enable TLS Encryption"
              type="switch"
              value={values.tls_enabled}
              disabled={isLoading}
              helperText="Requires a valid certificate and private key in the proxy core."
              onChange={onChange}
            />

            {values.tls_enabled && certOptions.length > 0 && (
              <FormField
                name="tls_cert_id"
                label={t('inbounds.certificate') || 'TLS Certificate'}
                type="select"
                value={values.tls_cert_id ? String(values.tls_cert_id) : ''}
                disabled={isLoading}
                options={[{ value: '', label: t('common.none') || '— None —' }, ...certOptions]}
                helperText={t('inbounds.certificateHelp') || 'Select a certificate to use for TLS. Managed in Certificates page.'}
                onChange={(name, value) => onChange(name, value ? Number(value) : 0)}
                onBlur={onBlur}
              />
            )}

            <FormField
              name="is_enabled"
              label="Enable Inbound Connection"
              type="switch"
              value={values.is_enabled}
              disabled={isLoading}
              onChange={onChange}
            />
          </CardContent>
        </Card>
      </div>

      {/* Footer Actions */}
      <div className="flex gap-3 justify-end pt-4 border-t border-border-primary mt-auto">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isLoading}
          className="w-24"
        >
          {t('common.cancel')}
        </Button>
        <Button
          type="submit"
          loading={isLoading}
          disabled={isLoading || isInvalid}
          className="min-w-[120px]"
        >
          {inbound ? <><Save className="w-4 h-4 mr-2" /> Save Changes</> : 'Create Inbound'}
        </Button>
      </div>
    </form>
  )
}
