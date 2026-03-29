import { useForm } from '../../hooks/useForm'
import { userSchema, UserFormData } from '../../utils/validators'
import { FormField } from './FormField'
import { Button } from '../ui/Button'
import { useCreateUser, useUpdateUser } from '../../hooks/useUsers'
import type { User } from '../../types'
import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '../ui/Card'
import { Save } from 'lucide-preact'

interface UserFormProps {
  user?: User
  onSuccess: () => void
  onCancel: () => void
}

export function UserForm({ user, onSuccess, onCancel }: UserFormProps) {
  const { t } = useTranslation()
  const isEdit = !!user

  const { mutate: createUser, isLoading: isCreating } = useCreateUser()
  const { mutate: updateUser, isLoading: isUpdating } = useUpdateUser()

  const {
    values,
    errors,
    touched,
    isSubmitting,
    handleChange,
    handleBlur,
    handleSubmit,
  } = useForm<UserFormData>({
    schema: userSchema,
    initialValues: user
      ? {
          username: user.username,
          email: user.email || '',
          traffic_limit_bytes: user.traffic_limit_bytes || undefined,
          expiry_date: user.expiry_date || '',
          is_active: user.is_active ?? true,
        }
      : {
          username: '',
          email: '',
          is_active: true,
        },
    onSubmit: async (data) => {
      if (isEdit && user) {
        await updateUser({ id: user.id, data: data as unknown as Record<string, unknown> })
      } else {
        await createUser(data as unknown as Record<string, unknown>)
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

            <FormField
              name="traffic_limit_bytes"
              label="Traffic Limit (Bytes)"
              type="number"
              value={values.traffic_limit_bytes}
              error={errors.traffic_limit_bytes}
              touched={touched.traffic_limit_bytes}
              disabled={isLoading}
              placeholder="Leave empty for unlimited"
              helperText="E.g. 107374182400 for 100GB"
              onChange={onChange}
              onBlur={onBlur}
            />

            <FormField
              name="expiry_date"
              label="Expiration Date"
              type="text"
              value={values.expiry_date}
              error={errors.expiry_date}
              touched={touched.expiry_date}
              disabled={isLoading}
              placeholder="YYYY-MM-DDTHH:MM:SSZ (optional)"
              helperText="Leave empty if the account never expires."
              onChange={onChange}
              onBlur={onBlur}
            />
          </CardContent>
        </Card>
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
