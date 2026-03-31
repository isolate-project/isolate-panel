import { useState, useEffect } from 'preact/hooks'
import { route } from 'preact-router'
import { Zap, ChevronLeft, ChevronRight, Check } from 'lucide-preact'

import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { FormField } from '../components/forms/FormField'
import { useCores } from '../hooks/useCores'
import { useProtocols, useProtocolSchema, useProtocolDefaults } from '../hooks/useProtocols'
import { useInbound, useUpdateInbound } from '../hooks/useInbounds'
import type { Core, ProtocolSummary } from '../types'

import { useTranslation } from 'react-i18next'

type WizardStep = 1 | 2 | 3 | 4 | 5

interface WizardData {
  core_id: number
  core_name: string
  protocol: string
  protocol_label: string
  name: string
  port: number
  listen_address: string
  config: Record<string, unknown>
  tls_enabled: boolean
  reality_enabled: boolean
  is_enabled: boolean
}

export function InboundEdit({ id }: { id: number }) {
  const { t } = useTranslation()
  const { data: inbound, isLoading: inboundLoading } = useInbound(id)
  const [step, setStep] = useState<WizardStep>(1)
  const [initialized, setInitialized] = useState(false)
  const [data, setData] = useState<WizardData>({
    core_id: 0,
    core_name: '',
    protocol: '',
    protocol_label: '',
    name: '',
    port: 443,
    listen_address: '0.0.0.0',
    config: {},
    tls_enabled: true,
    reality_enabled: false,
    is_enabled: true,
  })

  const { data: cores, isLoading: coresLoading } = useCores()
  const { data: protocolsData } = useProtocols(
    data.core_name ? { core: data.core_name, direction: 'inbound' } : undefined
  )
  const { data: schema } = useProtocolSchema(data.protocol)
  const { data: defaults } = useProtocolDefaults(data.protocol)
  const { mutate: updateInbound, isLoading: isUpdating } = useUpdateInbound()

  const allCores: Core[] = Array.isArray(cores) ? cores : []
  const protocols: ProtocolSummary[] = protocolsData?.protocols || []

  // Pre-fill wizard data from existing inbound
  useEffect(() => {
    if (inbound && allCores.length > 0 && !initialized) {
      let configObj: Record<string, unknown> = {}
      try {
        configObj = inbound.config_json ? JSON.parse(inbound.config_json) : {}
      } catch {
        // ignore
      }

      const core = allCores.find((c) => c.id === inbound.core_id)
      setData({
        core_id: inbound.core_id,
        core_name: core?.name || '',
        protocol: inbound.protocol,
        protocol_label: inbound.protocol.toUpperCase(),
        name: inbound.name,
        port: inbound.port,
        listen_address: inbound.listen_address,
        config: configObj,
        tls_enabled: inbound.tls_enabled,
        reality_enabled: inbound.reality_enabled,
        is_enabled: inbound.is_enabled,
      })
      setInitialized(true)
    }
  }, [inbound, allCores, initialized])

  // Update protocol label when protocols load
  useEffect(() => {
    if (initialized && protocols.length > 0 && data.protocol) {
      const proto = protocols.find((p) => p.protocol === data.protocol)
      if (proto && data.protocol_label === data.protocol.toUpperCase()) {
        setData((prev) => ({ ...prev, protocol_label: proto.label }))
      }
    }
  }, [protocols, initialized, data.protocol, data.protocol_label])

  const coreCards = [
    { name: 'singbox', label: 'Sing-box', desc: t('wizard.singboxDesc') },
    { name: 'xray', label: 'Xray', desc: t('wizard.xrayDesc') },
    { name: 'mihomo', label: 'Mihomo', desc: t('wizard.mihomoDesc') },
  ]

  const handleCoreSelect = (coreName: string) => {
    const core = allCores.find((c) => c.name === coreName)
    if (core) {
      setData((prev) => ({
        ...prev,
        core_id: core.id,
        core_name: core.name,
        protocol: '',
        protocol_label: '',
        config: {},
      }))
    }
  }

  const handleProtocolSelect = (protocol: string) => {
    const proto = protocols.find((p) => p.protocol === protocol)
    setData((prev) => ({
      ...prev,
      protocol,
      protocol_label: proto?.label || protocol,
      tls_enabled: proto?.requires_tls ?? prev.tls_enabled,
      config: {},
    }))
  }

  const handleConfigChange = (key: string, value: unknown) => {
    setData((prev) => ({
      ...prev,
      config: { ...prev.config, [key]: value },
    }))
  }

  const handleUpdate = async () => {
    const payload: Record<string, unknown> = {
      name: data.name,
      protocol: data.protocol,
      core_id: data.core_id,
      port: data.port,
      listen_address: data.listen_address,
      tls_enabled: data.tls_enabled,
      reality_enabled: data.reality_enabled,
      is_enabled: data.is_enabled,
      config_json: JSON.stringify(data.config),
    }

    try {
      await updateInbound({ id, data: payload })
      route(`/inbounds/${id}`)
    } catch {
      // Error handled by hook toast
    }
  }

  const canGoNext = (): boolean => {
    switch (step) {
      case 1: return data.core_id > 0
      case 2: return !!data.protocol
      case 3: return !!data.name && data.port > 0
      case 4: return true
      case 5: return true
      default: return false
    }
  }

  const stepLabels = [
    t('wizard.stepCore'),
    t('wizard.stepProtocol'),
    t('wizard.stepSettings'),
    t('wizard.stepTls'),
    t('wizard.stepReview'),
  ]

  if (inboundLoading || !initialized) {
    return (
      <PageLayout>
        <Card className="flex items-center justify-center py-12">
      <CardContent className="p-6">
          <Spinner size="lg" />
              </CardContent>
    </Card>
      </PageLayout>
    )
  }

  return (
    <PageLayout>
      <PageHeader
        title={t('wizard.editTitle')}
        description={t('wizard.editDescription')}
        actions={
          <Button variant="outline" onClick={() => route(`/inbounds/${id}`)}>
            {t('common.cancel')}
          </Button>
        }
      />

      {/* Step Indicator */}
      <Card className="mb-6">
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          {stepLabels.map((label, i) => {
            const stepNum = (i + 1) as WizardStep
            const isActive = step === stepNum
            const isDone = step > stepNum
            return (
              <div key={i} className="flex items-center gap-2 flex-1">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold shrink-0 ${
                  isDone ? 'bg-green-500 text-white' :
                  isActive ? 'bg-primary text-white' :
                  'bg-secondary text-tertiary'
                }`}>
                  {isDone ? <Check className="w-4 h-4" /> : stepNum}
                </div>
                <span className={`text-sm hidden md:inline ${isActive ? 'text-primary font-medium' : 'text-tertiary'}`}>
                  {label}
                </span>
                {i < 4 && <div className="flex-1 h-px bg-primary mx-2 hidden md:block" />}
              </div>
            )
          })}
        </div>
            </CardContent>
    </Card>

      {/* Step Content */}
      <Card className="mb-6">
      <CardContent className="p-6">
        {/* Step 1: Choose Core */}
        {step === 1 && (
          <div>
            <h3 className="text-lg font-semibold text-primary mb-4">{t('wizard.chooseCoreTitle')}</h3>
            {coresLoading ? (
              <div className="flex justify-center py-8"><Spinner size="lg" /></div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {coreCards.map((card) => {
                  const core = allCores.find((c) => c.name === card.name)
                  const isSelected = data.core_name === card.name
                  return (
                    <button
                      key={card.name}
                      className={`p-6 rounded-lg border-2 text-left transition-base ${
                        isSelected
                          ? 'border-blue-500 bg-blue-50 dark:bg-blue-950'
                          : 'border-primary hover:border-blue-300'
                      }`}
                      onClick={() => handleCoreSelect(card.name)}
                    >
                      <div className="flex items-center justify-between mb-2">
                        <h4 className="font-bold text-primary">{card.label}</h4>
                        {core && (
                          <Badge variant={core.is_running ? 'success' : 'default'}>
                            {core.is_running ? t('cores.running') : t('cores.stopped')}
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-secondary">{card.desc}</p>
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        )}

        {/* Step 2: Choose Protocol */}
        {step === 2 && (
          <div>
            <h3 className="text-lg font-semibold text-primary mb-4">{t('wizard.chooseProtocolTitle')}</h3>
            {protocols.length === 0 ? (
              <p className="text-secondary py-8 text-center">{t('wizard.noProtocols')}</p>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {protocols.map((proto) => {
                  const isSelected = data.protocol === proto.protocol
                  return (
                    <button
                      key={proto.protocol}
                      className={`p-4 rounded-lg border-2 text-left transition-base ${
                        isSelected
                          ? 'border-blue-500 bg-blue-50 dark:bg-blue-950'
                          : 'border-primary hover:border-blue-300'
                      }`}
                      onClick={() => handleProtocolSelect(proto.protocol)}
                    >
                      <div className="flex items-center gap-2 mb-1">
                        <h4 className="font-bold text-primary">{proto.label}</h4>
                        {proto.requires_tls && <Badge variant="outline" className="text-xs">TLS</Badge>}
                      </div>
                      <p className="text-xs text-secondary">{proto.description}</p>
                      <Badge variant="default" className="mt-2 text-xs">{proto.category}</Badge>
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        )}

        {/* Step 3: Protocol Settings */}
        {step === 3 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-primary mb-4">{t('wizard.settingsTitle')}</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-primary mb-1">{t('inbounds.name')}</label>
                <Input
                  value={data.name}
                  onChange={(e: Event) => setData((prev) => ({ ...prev, name: (e.target as HTMLInputElement).value }))}
                  placeholder={t('wizard.namePlaceholder')}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-primary mb-1">{t('inbounds.port')}</label>
                <Input
                  type="number"
                  value={data.port.toString()}
                  onChange={(e: Event) => setData((prev) => ({ ...prev, port: Number((e.target as HTMLInputElement).value) }))}
                  placeholder="443"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-primary mb-1">{t('inbounds.listenAddress')}</label>
                <Input
                  value={data.listen_address}
                  onChange={(e: Event) => setData((prev) => ({ ...prev, listen_address: (e.target as HTMLInputElement).value }))}
                  placeholder="0.0.0.0"
                />
              </div>
            </div>

            {/* Dynamic protocol fields from schema */}
            {schema?.parameters && Object.keys(schema.parameters).length > 0 && (
              <div className="mt-6">
                <h4 className="text-sm font-semibold text-primary mb-3">{t('wizard.protocolSettings')}</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {Object.entries(schema.parameters)
                    .filter(([, param]) => param.group === 'basic' || !param.group)
                    .map(([key, param]) => (
                      <div key={key}>
                        <FormField
                          name={key}
                          label={param.label}
                          type={param.type === 'boolean' ? 'switch' :
                                param.type === 'select' ? 'select' :
                                param.type === 'integer' ? 'number' : 'text'}
                          value={data.config[key] as string | number | boolean ?? param.default as string | number | boolean ?? ''}
                          options={param.options?.map((o) => ({ value: o, label: o }))}
                          placeholder={param.example}
                          helperText={param.description}
                          onChange={(_, value) => handleConfigChange(key, value)}
                        />
                        {param.auto_generate && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="mt-1"
                            onClick={() => {
                              if (defaults?.defaults?.[key]) {
                                handleConfigChange(key, defaults.defaults[key])
                              }
                            }}
                          >
                            <Zap className="w-3 h-3 mr-1" />
                            {t('wizard.autoGenerate')}
                          </Button>
                        )}
                      </div>
                    ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Step 4: TLS / Transport */}
        {step === 4 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-primary mb-4">{t('wizard.tlsTitle')}</h3>
            <FormField
              name="tls_enabled"
              label="TLS"
              type="switch"
              value={data.tls_enabled}
              onChange={(_, value) => setData((prev) => ({ ...prev, tls_enabled: value as boolean }))}
            />
            <FormField
              name="reality_enabled"
              label="REALITY"
              type="switch"
              value={data.reality_enabled}
              onChange={(_, value) => setData((prev) => ({ ...prev, reality_enabled: value as boolean }))}
            />
            <FormField
              name="is_enabled"
              label={t('inbounds.enableInbound')}
              type="switch"
              value={data.is_enabled}
              onChange={(_, value) => setData((prev) => ({ ...prev, is_enabled: value as boolean }))}
            />

            {/* Transport settings from schema */}
            {schema?.transport && schema.transport.length > 0 && (
              <div className="mt-4">
                <label className="block text-sm font-medium text-primary mb-1">{t('wizard.transport')}</label>
                <Select
                  value={(data.config.transport as string) || 'tcp'}
                  onChange={(e: Event) => handleConfigChange('transport', (e.target as HTMLSelectElement).value)}
                  options={schema.transport.map((tr) => ({ value: tr, label: tr.toUpperCase() }))}
                />
              </div>
            )}
          </div>
        )}

        {/* Step 5: Review & Update */}
        {step === 5 && (
          <div>
            <h3 className="text-lg font-semibold text-primary mb-4">{t('wizard.reviewTitle')}</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-2 text-sm">
                <span className="text-secondary">{t('inbounds.core')}:</span>
                <span className="text-primary font-medium">{data.core_name}</span>
                <span className="text-secondary">{t('inbounds.protocol')}:</span>
                <span className="text-primary font-medium">{data.protocol_label}</span>
                <span className="text-secondary">{t('inbounds.name')}:</span>
                <span className="text-primary font-medium">{data.name}</span>
                <span className="text-secondary">{t('inbounds.port')}:</span>
                <span className="text-primary font-medium">{data.port}</span>
                <span className="text-secondary">{t('inbounds.listenAddress')}:</span>
                <span className="text-primary font-medium">{data.listen_address}</span>
                <span className="text-secondary">TLS:</span>
                <span className="text-primary font-medium">{data.tls_enabled ? t('common.yes') : t('common.no')}</span>
                <span className="text-secondary">REALITY:</span>
                <span className="text-primary font-medium">{data.reality_enabled ? t('common.yes') : t('common.no')}</span>
              </div>
              {Object.keys(data.config).length > 0 && (
                <div className="mt-4">
                  <span className="text-sm text-secondary">{t('wizard.configPreview')}:</span>
                  <pre className="mt-1 p-3 bg-secondary rounded text-xs text-primary overflow-x-auto">
                    {JSON.stringify(data.config, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        )}
            </CardContent>
    </Card>

      {/* Navigation Buttons */}
      <div className="flex justify-between">
        <Button
          variant="outline"
          onClick={() => setStep((s) => Math.max(1, s - 1) as WizardStep)}
          disabled={step === 1}
        >
          <ChevronLeft className="w-4 h-4 mr-1" />
          {t('wizard.back')}
        </Button>

        {step < 5 ? (
          <Button
            variant="default"
            onClick={() => setStep((s) => Math.min(5, s + 1) as WizardStep)}
            disabled={!canGoNext()}
          >
            {t('wizard.next')}
            <ChevronRight className="w-4 h-4 ml-1" />
          </Button>
        ) : (
          <Button
            variant="default"
            onClick={handleUpdate}
            loading={isUpdating}
            disabled={isUpdating}
          >
            <Check className="w-4 h-4 mr-1" />
            {t('wizard.updateInbound')}
          </Button>
        )}
      </div>
    </PageLayout>
  )
}
