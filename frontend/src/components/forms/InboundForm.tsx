import { useState, useEffect } from 'preact/hooks'
import { useForm } from '../../hooks/useForm'
import { inboundSchema, InboundFormData } from '../../utils/validators'
import { FormField } from './FormField'
import { Button } from '../ui/Button'
import { useCreateInbound, useUpdateInbound } from '../../hooks/useInbounds'
import { useCores } from '../../hooks/useCores'
import type { Inbound, Core } from '../../types'
import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '../ui/Card'
import { Save } from 'lucide-preact'

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
        }
      : {
          name: '',
          protocol: 'vless' as const,
          port: 443,
          core_id: Number(coreOptions[0]?.value || '1'),
          listen_address: '0.0.0.0',
          is_enabled: true,
          tls_enabled: true,
        },
    onSubmit: async (data) => {
      const payload = data as unknown as Record<string, unknown>
      if (inbound) {
        await updateInbound({ id: inbound.id, data: payload })
      } else {
        await createInbound(payload)
      }
      onSuccess()
    },
  })

  const isLoading = Boolean(isSubmitting || isCreating || isUpdating)

  const onChange = (name: string, value: string | number | boolean) => {
    handleChange(name as keyof InboundFormData, value)
    
    // Update core type when core changes
    if (name === 'core_id') {
      const selectedCore = coreOptions.find(c => c.value === String(value))
      if (selectedCore?.type) {
        setSelectedCoreType(selectedCore.type)
        // Reset protocol to first supported if current is not supported
        const supportedProtocols = CORE_PROTOCOLS[selectedCore.type] || []
        if (!supportedProtocols.includes(values.protocol)) {
          handleChange('protocol', supportedProtocols[0] || 'vless')
        }
      }
    }
  }
  
  const onBlur = (name: string) => {
    handleBlur(name as keyof InboundFormData)
  }

  // Initialize core type on mount
  useEffect(() => {
    if (inbound && cores) {
      const core = cores.find(c => c.id === inbound.core_id)
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
                onChange={onChange}
                onBlur={onBlur}
              />
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
          disabled={isLoading}
          className="min-w-[120px]"
        >
          {inbound ? <><Save className="w-4 h-4 mr-2" /> Save Changes</> : 'Create Inbound'}
        </Button>
      </div>
    </form>
  )
}
