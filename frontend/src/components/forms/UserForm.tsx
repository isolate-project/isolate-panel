import { useForm } from '../../hooks/useForm'
import { userSchema, UserFormData } from '../../utils/validators'
import { FormField } from './FormField'
import { Button } from '../ui/Button'
import { useCreateUser, useUpdateUser } from '../../hooks/useUsers'
import type { User } from '../../types'
import { useTranslation } from 'react-i18next'

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
          expiry_date: user.expire_at || '',
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

  // Widen the typed handleChange/handleBlur for FormField compatibility
  const onChange = (name: string, value: string | number | boolean) => {
    handleChange(name as keyof UserFormData, value)
  }
  const onBlur = (name: string) => {
    handleBlur(name as keyof UserFormData)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <FormField
        name="username"
        label={t('users.username')}
        type="text"
        value={values.username}
        error={errors.username}
        touched={touched.username}
        required
        disabled={isLoading}
        placeholder="john_doe"
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
        placeholder="user@example.com"
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="traffic_limit_bytes"
        label={t('users.trafficLimit') + ' (bytes)'}
        type="number"
        value={values.traffic_limit_bytes}
        error={errors.traffic_limit_bytes}
        touched={touched.traffic_limit_bytes}
        disabled={isLoading}
        placeholder="107374182400"
        helperText={t('users.trafficLimitHint')}
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="expiry_date"
        label={t('users.expiryDate')}
        type="text"
        value={values.expiry_date}
        error={errors.expiry_date}
        touched={touched.expiry_date}
        disabled={isLoading}
        placeholder="2025-12-31T23:59:59Z"
        helperText={t('users.expiryDateHint')}
        onChange={onChange}
        onBlur={onBlur}
      />

      <FormField
        name="is_active"
        label={t('users.isActive')}
        type="switch"
        value={values.is_active}
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
          {isEdit ? t('common.save') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}
