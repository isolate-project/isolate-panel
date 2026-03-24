import { useForm } from '../../hooks/useForm'
import { inboundSchema, InboundFormData } from '../../utils/validators'
import { FormField } from './FormField'
import { Button } from '../ui/Button'
import { useCreateInbound, useUpdateInbound } from '../../hooks/useInbounds'
import { useCores } from '../../hooks/useCores'
import type { Inbound, Core } from '../../types'
import { useTranslation } from 'react-i18next'

interface InboundFormProps {
  inbound?: Inbound
  onSuccess: () => void
  onCancel: () => void
}

export function InboundForm({ inbound, onSuccess, onCancel }: InboundFormProps) {
  const { t } = useTranslation()
  const { mutate: createInbound, isLoading: isCreating } = useCreateInbound()
  const { mutate: updateInbound, isLoading: isUpdating } = useUpdateInbound()
  const { data: cores } = useCores()

  const coreOptions = Array.isArray(cores)
    ? cores.map((c: Core) => ({ value: c.id.toString(), label: `${c.name} (${c.type})` }))
    : [
        { value: '1', label: 'Sing-box' },
        { value: '2', label: 'Xray' },
        { value: '3', label: 'Mihomo' },
      ]

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

  const isLoading = isSubmitting || isCreating || isUpdating

  // Widen typed handleChange/handleBlur for FormField compatibility
  const onChange = (name: string, value: string | number | boolean) => {
    handleChange(name as keyof InboundFormData, value)
  }
  const onBlur = (name: string) => {
    handleBlur(name as keyof InboundFormData)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <FormField
        name="name"
        label={t('inbounds.name')}
        type="text"
        value={values.name}
        error={errors.name}
        touched={touched.name}
        required
        disabled={isLoading}
        placeholder="VLESS-443"
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="protocol"
        label={t('inbounds.protocol')}
        type="select"
        value={values.protocol}
        error={errors.protocol}
        touched={touched.protocol}
        required
        disabled={isLoading}
        options={[
          { value: 'vless', label: 'VLESS' },
          { value: 'vmess', label: 'VMess' },
          { value: 'trojan', label: 'Trojan' },
          { value: 'shadowsocks', label: 'Shadowsocks' },
          { value: 'hysteria2', label: 'Hysteria2' },
          { value: 'tuic', label: 'TUIC' },
        ]}
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
        onChange={(name, value) => onChange(name, Number(value))}
        onBlur={onBlur}
      />

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

      <FormField
        name="tls_enabled"
        label="TLS"
        type="switch"
        value={values.tls_enabled}
        disabled={isLoading}
        onChange={onChange}
      />

      <FormField
        name="is_enabled"
        label={t('inbounds.enableInbound')}
        type="switch"
        value={values.is_enabled}
        disabled={isLoading}
        onChange={onChange}
      />

      <div className="flex gap-3 justify-end pt-4 border-t border-primary">
        <Button
          type="button"
          variant="secondary"
          onClick={onCancel}
          disabled={isLoading}
        >
          {t('common.cancel')}
        </Button>
        <Button
          type="submit"
          variant="primary"
          loading={isLoading}
          disabled={isLoading}
        >
          {inbound ? t('common.save') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}
