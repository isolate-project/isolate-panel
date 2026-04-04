import { useState, useCallback, useRef } from 'preact/hooks'
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

  // Store schema, values, and onSubmit in refs to stabilize callbacks
  const schemaRef = useRef(schema)
  schemaRef.current = schema
  const valuesRef = useRef(values)
  valuesRef.current = values
  const onSubmitRef = useRef(onSubmit)
  onSubmitRef.current = onSubmit
  const initialValuesRef = useRef(initialValues)
  initialValuesRef.current = initialValues

  const validate = useCallback((): boolean => {
    try {
      schemaRef.current.parse(valuesRef.current)
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
  }, [])

  const handleChange = useCallback((name: keyof T, value: string | number | boolean | undefined) => {
    setValues((prev) => ({ ...prev, [name]: value }))
    
    // Clear error for this field when user starts typing
    setErrors((prev) => {
      if (prev[name as string]) {
        const newErrors = { ...prev }
        delete newErrors[name as string]
        return newErrors
      }
      return prev
    })
  }, [])

  const handleBlur = useCallback((name: keyof T) => {
    setTouched((prev) => ({ ...prev, [name as string]: true }))
    
    // Validate single field on blur - simplified approach
    try {
      schemaRef.current.parse(valuesRef.current)
      setErrors((prev) => {
        if (prev[name as string]) {
          const newErrors = { ...prev }
          delete newErrors[name as string]
          return newErrors
        }
        return prev
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
  }, [])

  const handleSubmit = useCallback(async (e?: Event) => {
    e?.preventDefault()
    
    if (!validate()) {
      // Mark all fields as touched to show errors
      const allTouched: Record<string, boolean> = {}
      Object.keys(valuesRef.current).forEach((key) => {
        allTouched[key] = true
      })
      setTouched(allTouched)
      return
    }

    setIsSubmitting(true)
    try {
      await onSubmitRef.current(valuesRef.current as T)
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Submission failed'
      setErrors(prev => ({ ...prev, _form: message }))
    } finally {
      setIsSubmitting(false)
    }
  }, [validate])

  const setFieldValue = useCallback((name: keyof T, value: string | number | boolean | undefined) => {
    setValues((prev) => ({ ...prev, [name]: value }))
  }, [])

  const setFieldError = useCallback((name: keyof T, error: string) => {
    setErrors((prev) => ({ ...prev, [name as string]: error }))
  }, [])

  const reset = useCallback(() => {
    setValues(initialValuesRef.current)
    setErrors({})
    setTouched({})
    setIsSubmitting(false)
  }, [])

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
