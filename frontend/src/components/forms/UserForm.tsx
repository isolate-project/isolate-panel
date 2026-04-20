import { useState, useEffect, useMemo } from 'preact/hooks'
import { useForm } from '../../hooks/useForm'
import { userSchema, UserFormData } from '../../utils/validators'
import { FormField } from './FormField'
import { Button } from '../ui/Button'
import { useCreateUser, useUpdateUser } from '../../hooks/useUsers'
import { useInbounds } from '../../hooks/useInbounds'
import type { User, Inbound } from '../../types'
import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '../ui/Card'
import { Save } from 'lucide-preact'

interface UserFormProps {
  user?: User
  userInboundIds?: number[]
  onSuccess: () => void
  onCancel: () => void
}

export function UserForm({ user, userInboundIds, onSuccess, onCancel }: UserFormProps) {
  const { t } = useTranslation()
  const isEdit = !!user

  const { mutate: createUser, isLoading: isCreating } = useCreateUser()
  const { mutate: updateUser, isLoading: isUpdating } = useUpdateUser()
  const { data: inboundsData } = useInbounds()

  const [selectedInboundIds, setSelectedInboundIds] = useState<number[]>(userInboundIds || [])
  const [trafficUnit, setTrafficUnit] = useState<'GB' | 'MB'>(
    user?.traffic_limit_bytes && user.traffic_limit_bytes < 1073741824 ? 'MB' : 'GB'
  )

  useEffect(() => {
    if (userInboundIds) setSelectedInboundIds(userInboundIds)
  }, [userInboundIds])

  const allInbounds: Inbound[] = Array.isArray(inboundsData) ? inboundsData : inboundsData?.items || []

  const GB = 1073741824
  const MB = 1048576
  const unitMultiplier = trafficUnit === 'GB' ? GB : MB

  const bytesToDisplay = useMemo(() => {
    const bytes = user?.traffic_limit_bytes
    if (!bytes) return undefined
    if (bytes >= GB && bytes % GB === 0) return bytes / GB
    return bytes / MB
  }, [user?.traffic_limit_bytes])


  const computeExpiryDays = (u: User): number | undefined => {
    if (!u.expiry_date) return undefined
    const diff = new Date(u.expiry_date).getTime() - Date.now()
    return diff > 0 ? Math.ceil(diff / (1000 * 60 * 60 * 24)) : 1
  }

  const toggleInbound = (id: number) => {
    setSelectedInboundIds((prev) =>
      prev.includes(id) ? prev.filter((i) => i !== id) : [...prev, id]
    )
  }

  const {
    values,
    errors,
    touched,
    isSubmitting,
    handleChange,
    handleBlur,
    handleSubmit,
  } = useForm({
    schema: userSchema,
    initialValues: user
      ? {
          username: user.username,
          email: user.email || '',
          traffic_limit_bytes: bytesToDisplay,
          expiry_days: computeExpiryDays(user),
          unlimited: !user.expiry_date,
          is_active: user.is_active ?? true,
        }
      : {
          username: '',
          email: '',
          unlimited: true,
          is_active: true,
        },
    onSubmit: async (data) => {
      const trafficBytes = data.traffic_limit_bytes
        ? data.traffic_limit_bytes * unitMultiplier
        : undefined
      const payload: Record<string, unknown> = {
        username: data.username,
        email: data.email,
        is_active: data.is_active,
        traffic_limit_bytes: trafficBytes,
        expiry_days: data.unlimited ? null : data.expiry_days,
        inbound_ids: selectedInboundIds,
      }
      if (isEdit && user) {
        await updateUser({ id: user.id, data: payload })
      } else {
        await createUser(payload)
      }
      onSuccess()
    },
  })

  const isLoading = isSubmitting || isCreating || isUpdating

  const onChange = (name: string, value: string | number | boolean) => {
    handleChange(name as keyof UserFormData, value)
  }
  const onBlur = (name: string) => {
    handleBlur(name as keyof UserFormData)
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col h-full">
      <div className="flex-1 space-y-6 pb-6">
        <Card className="border border-border-primary/50 bg-bg-secondary/30 shadow-none">
          <CardContent className="p-5 space-y-5">
            <h3 className="font-semibold text-text-primary text-sm mb-4 uppercase tracking-wider">Account Details</h3>

            <FormField
              name="username"
              label={t('users.username')}
              type="text"
              value={values.username}
              error={errors.username}
              touched={touched.username}
              required
              disabled={isLoading || isEdit}
              placeholder="e.g. john_doe"
              onChange={onChange}
              onBlur={onBlur}
            />

            <FormField
              name="email"
              label={t('users.email')}
              type="email"
              value={values.email}
              error={errors.email}
              touched={touched.email}
              disabled={isLoading}
              placeholder="user@example.com (optional)"
              onChange={onChange}
              onBlur={onBlur}
            />

            <FormField
              name="is_active"
              label="Enable User Account"
              type="switch"
              value={values.is_active}
              disabled={isLoading}
              helperText="If disabled, the user will be instantly disconnected and blocked."
              onChange={onChange}
            />
          </CardContent>
        </Card>

        <Card className="border border-border-primary/50 bg-bg-secondary/30 shadow-none">
          <CardContent className="p-5 space-y-5">
            <h3 className="font-semibold text-text-primary text-sm mb-4 uppercase tracking-wider">Limits & Expiration</h3>

            <div className="space-y-2">
              <label className="text-sm font-medium text-text-primary">Traffic Limit</label>
              <div className="flex gap-2">
                <div className="flex-1">
                  <FormField
                    name="traffic_limit_bytes"
                    label=""
                    type="number"
                    value={values.traffic_limit_bytes}
                    error={errors.traffic_limit_bytes}
                    touched={touched.traffic_limit_bytes}
                    disabled={isLoading}
                    placeholder="Leave empty for unlimited"
                    onChange={onChange}
                    onBlur={onBlur}
                  />
                </div>
                <div className="flex rounded-lg border border-border-primary overflow-hidden self-start">
                  <button
                    type="button"
                    onClick={() => setTrafficUnit('GB')}
                    className={`px-3 py-2 text-sm font-medium transition-base ${
                      trafficUnit === 'GB'
                        ? 'bg-primary text-text-inverse'
                        : 'bg-bg-primary text-text-secondary hover:bg-bg-hover'
                    }`}
                  >GB</button>
                  <button
                    type="button"
                    onClick={() => setTrafficUnit('MB')}
                    className={`px-3 py-2 text-sm font-medium transition-base ${
                      trafficUnit === 'MB'
                        ? 'bg-primary text-text-inverse'
                        : 'bg-bg-primary text-text-secondary hover:bg-bg-hover'
                    }`}
                  >MB</button>
                </div>
              </div>
              <p className="text-xs text-text-secondary">
                {values.traffic_limit_bytes
                  ? `= ${(values.traffic_limit_bytes * unitMultiplier / GB).toFixed(values.traffic_limit_bytes * unitMultiplier >= GB ? 0 : 2)} GB`
                  : 'Leave empty for unlimited traffic'}
              </p>
            </div>

            <FormField
              name="unlimited"
              label="Unlimited Subscription"
              type="switch"
              value={values.unlimited}
              disabled={isLoading}
              helperText="If enabled, the subscription never expires."
              onChange={onChange}
            />

            {!values.unlimited && (
              <FormField
                name="expiry_days"
                label="Subscription Duration (days)"
                type="number"
                value={values.expiry_days}
                error={errors.expiry_days}
                touched={touched.expiry_days}
                disabled={isLoading}
                placeholder="e.g. 30"
                helperText="Number of days from now until expiration."
                onChange={onChange}
                onBlur={onBlur}
              />
            )}
          </CardContent>
        </Card>

        {allInbounds.length > 0 && (
          <Card className="border border-border-primary/50 bg-bg-secondary/30 shadow-none">
            <CardContent className="p-5 space-y-4">
              <h3 className="font-semibold text-text-primary text-sm mb-4 uppercase tracking-wider">Inbounds</h3>
              <p className="text-xs text-text-secondary">Select inbounds this user should have access to.</p>
              <div className="space-y-2 max-h-48 overflow-y-auto">
                {allInbounds.map((inbound: Inbound) => {
                  const isSelected = selectedInboundIds.includes(inbound.id)
                  return (
                    <button
                      key={inbound.id}
                      type="button"
                      onClick={() => toggleInbound(inbound.id)}
                      disabled={isLoading}
                      className={`w-full flex items-center justify-between p-3 rounded-lg border text-left text-sm transition-base ${
                        isSelected
                          ? 'border-primary bg-primary/5 text-text-primary'
                          : 'border-border-primary bg-bg-primary text-text-secondary hover:border-primary/50'
                      } disabled:opacity-50`}
                    >
                      <div>
                        <span className="font-medium">{inbound.name}</span>
                        <span className="ml-2 text-xs text-text-tertiary">{inbound.protocol} :{inbound.port}</span>
                      </div>
                      <div className={`w-4 h-4 rounded border-2 flex items-center justify-center ${
                        isSelected ? 'border-primary bg-primary' : 'border-border-secondary'
                      }`}>
                        {isSelected && (
                          <svg className="w-3 h-3 text-text-inverse" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                          </svg>
                        )}
                      </div>
                    </button>
                  )
                })}
              </div>
              <p className="text-xs text-text-tertiary">
                {selectedInboundIds.length} inbound{selectedInboundIds.length !== 1 ? 's' : ''} selected
              </p>
            </CardContent>
          </Card>
        )}
      </div>

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
          {isEdit ? <><Save className="w-4 h-4 mr-2" /> Save Changes</> : 'Create User'}
        </Button>
      </div>
    </form>
  )
}
