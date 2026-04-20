import { useForm } from '../../hooks/useForm'
import { FormField } from './FormField'
import { Button } from '../ui/Button'
import { useCreateOutbound, useUpdateOutbound } from '../../hooks/useOutbounds'
import { useCores } from '../../hooks/useCores'
import { useProtocols } from '../../hooks/useProtocols'
import type { Outbound, Core, ProtocolSummary } from '../../types'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

const outboundSchema = z.object({
  name: z.string().min(1, { message: 'validation.nameRequired' }).max(100, { message: 'validation.nameTooLong' }),
  protocol: z.string().min(1, { message: 'validation.nameRequired' }),
  core_id: z.number().min(1, { message: 'validation.coreRequired' }),
  config_json: z.string().default('{}').pipe(z.string()),
  priority: z.number().default(0).pipe(z.number()),
  is_enabled: z.boolean().default(true).pipe(z.boolean()),
})

type OutboundFormData = z.infer<typeof outboundSchema>

interface OutboundFormProps {
  outbound?: Outbound
  onSuccess: () => void
  onCancel: () => void
}

export function OutboundForm({ outbound, onSuccess, onCancel }: OutboundFormProps) {
  const { t } = useTranslation()
  const { mutate: createOutbound, isLoading: isCreating } = useCreateOutbound()
  const { mutate: updateOutbound, isLoading: isUpdating } = useUpdateOutbound()
  const { data: cores } = useCores()
  const { data: protocolsData } = useProtocols({ direction: 'outbound' })

  const coreOptions = Array.isArray(cores)
    ? cores.map((c: Core) => ({ value: c.id.toString(), label: `${c.name}` }))
    : [
        { value: '1', label: 'Sing-box' },
        { value: '2', label: 'Xray' },
        { value: '3', label: 'Mihomo' },
      ]

  const protocolOptions = protocolsData?.protocols
    ? protocolsData.protocols.map((p: ProtocolSummary) => ({ value: p.protocol, label: p.label }))
    : [
        { value: 'direct', label: 'Direct' },
        { value: 'block', label: 'Block' },
        { value: 'vless', label: 'VLESS' },
        { value: 'vmess', label: 'VMess' },
        { value: 'trojan', label: 'Trojan' },
        { value: 'shadowsocks', label: 'Shadowsocks' },
      ]

  const {
    values,
    errors,
    touched,
    isSubmitting,
    handleChange,
    handleBlur,
    handleSubmit,
  } = useForm({
    schema: outboundSchema,
    initialValues: outbound
      ? {
          name: outbound.name,
          protocol: outbound.protocol,
          core_id: outbound.core_id,
          config_json: outbound.config_json || '{}',
          priority: outbound.priority ?? 0,
          is_enabled: outbound.is_enabled ?? true,
        }
      : {
          name: '',
          protocol: 'direct',
          core_id: Number(coreOptions[0]?.value || '1'),
          config_json: '{}',
          priority: 0,
          is_enabled: true,
        },
    onSubmit: async (data) => {
      const payload = data as unknown as Record<string, unknown>
      if (outbound) {
        await updateOutbound({ id: outbound.id, data: payload })
      } else {
        await createOutbound(payload)
      }
      onSuccess()
    },
  })

  const isLoading = isSubmitting || isCreating || isUpdating

  const onChange = (name: string, value: string | number | boolean) => {
    handleChange(name as keyof OutboundFormData, value)
  }
  const onBlur = (name: string) => {
    handleBlur(name as keyof OutboundFormData)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <FormField
        name="name"
        label={t('outbounds.name')}
        type="text"
        value={values.name}
        error={errors.name}
        touched={touched.name}
        required
        disabled={isLoading}
        placeholder="direct-out"
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="protocol"
        label={t('outbounds.protocol')}
        type="select"
        value={values.protocol}
        error={errors.protocol}
        touched={touched.protocol}
        required
        disabled={isLoading}
        options={protocolOptions}
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="core_id"
        label={t('outbounds.core')}
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
        name="priority"
        label={t('outbounds.priority')}
        type="number"
        value={values.priority}
        error={errors.priority}
        touched={touched.priority}
        disabled={isLoading}
        placeholder="0"
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="config_json"
        label={t('outbounds.configJson')}
        type="text"
        value={values.config_json}
        error={errors.config_json}
        touched={touched.config_json}
        disabled={isLoading}
        placeholder="{}"
        helperText={t('outbounds.configJsonHint')}
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="is_enabled"
        label={t('outbounds.enableOutbound')}
        type="switch"
        value={values.is_enabled}
        disabled={isLoading}
        onChange={onChange}
      />

      <div className="flex gap-3 justify-end pt-4 border-t border-primary">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isLoading}
        >
          {t('common.cancel')}
        </Button>
        <Button
          type="submit"
          variant="default"
          loading={isLoading}
          disabled={isLoading}
        >
          {outbound ? t('common.save') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}
