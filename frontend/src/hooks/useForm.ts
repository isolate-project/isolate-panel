import { useState, useCallback } from 'preact/hooks'
import { z } from 'zod'

interface UseFormOptions<T> {
  schema: z.ZodSchema<T>
  onSubmit: (data: T) => Promise<void> | void
  initialValues?: Partial<T>
}

interface UseFormReturn<T> {
  values: Partial<T>
  errors: Record<string, string>
  isSubmitting: boolean
  touched: Record<string, boolean>
  handleChange: (name: keyof T, value: string | number | boolean | undefined) => void
  handleBlur: (name: keyof T) => void
  handleSubmit: (e?: Event) => Promise<void>
  setFieldValue: (name: keyof T, value: string | number | boolean | undefined) => void
  setFieldError: (name: keyof T, error: string) => void
  setValues: (values: Partial<T>) => void
  reset: () => void
  validate: () => boolean
}

export function useForm<T extends Record<string, unknown>>({
  schema,
  onSubmit,
  initialValues = {},
}: UseFormOptions<T>): UseFormReturn<T> {
  const [values, setValues] = useState<Partial<T>>(initialValues)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [touched, setTouched] = useState<Record<string, boolean>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  const validate = useCallback((): boolean => {
    try {
      schema.parse(values)
      setErrors({})
      return true
    } catch (error) {
      if (error instanceof z.ZodError) {
        const newErrors: Record<string, string> = {}
        error.issues.forEach((issue: z.ZodIssue) => {
          if (issue.path[0]) {
            newErrors[issue.path[0] as string] = issue.message
          }
        })
        setErrors(newErrors)
      }
      return false
    }
  }, [schema, values])

  const handleChange = useCallback((name: keyof T, value: string | number | boolean | undefined) => {
    setValues((prev) => ({ ...prev, [name]: value }))
    
    // Clear error for this field when user starts typing
    if (errors[name as string]) {
      setErrors((prev) => {
        const newErrors = { ...prev }
        delete newErrors[name as string]
        return newErrors
      })
    }
  }, [errors])

  const handleBlur = useCallback((name: keyof T) => {
    setTouched((prev) => ({ ...prev, [name as string]: true }))
    
    // Validate single field on blur - simplified approach
    try {
      schema.parse(values)
      setErrors((prev) => {
        const newErrors = { ...prev }
        delete newErrors[name as string]
        return newErrors
      })
    } catch (error) {
      if (error instanceof z.ZodError) {
        const fieldError = error.issues.find((issue: z.ZodIssue) => issue.path[0] === name)
        if (fieldError) {
          setErrors((prev) => ({
            ...prev,
            [name as string]: fieldError.message,
          }))
        }
      }
    }
  }, [schema, values])

  const handleSubmit = useCallback(async (e?: Event) => {
    e?.preventDefault()
    
    if (!validate()) {
      // Mark all fields as touched to show errors
      const allTouched: Record<string, boolean> = {}
      Object.keys(values).forEach((key) => {
        allTouched[key] = true
      })
      setTouched(allTouched)
      return
    }

    setIsSubmitting(true)
    try {
      await onSubmit(values as T)
    } catch (error) {
      console.error('Form submission error:', error)
    } finally {
      setIsSubmitting(false)
    }
  }, [validate, values, onSubmit])

  const setFieldValue = useCallback((name: keyof T, value: string | number | boolean | undefined) => {
    setValues((prev) => ({ ...prev, [name]: value }))
  }, [])

  const setFieldError = useCallback((name: keyof T, error: string) => {
    setErrors((prev) => ({ ...prev, [name as string]: error }))
  }, [])

  const reset = useCallback(() => {
    setValues(initialValues)
    setErrors({})
    setTouched({})
    setIsSubmitting(false)
  }, [initialValues])

  return {
    values,
    errors,
    isSubmitting,
    touched,
    handleChange,
    handleBlur,
    handleSubmit,
    setFieldValue,
    setFieldError,
    setValues,
    reset,
    validate,
  }
}
