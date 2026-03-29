import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/preact'
import { useForm } from './useForm'
import { z } from 'zod'

describe('useForm Hook', () => {
  const schema = z.object({
    username: z.string().min(3),
    email: z.string().email(),
  })

  const initialValues = {
    username: '',
    email: '',
  }

  const mockSubmit = vi.fn().mockResolvedValue({ success: true })

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should initialize with initial values', () => {
    const { result } = renderHook(() => useForm({ initialValues, schema }))
    expect(result.current.values).toEqual(initialValues)
    expect(result.current.errors).toEqual({})
  })

  it('should update values on change', () => {
    const { result } = renderHook(() => useForm({ initialValues, schema }))
    
    act(() => {
      result.current.handleChange('username', 'testuser')
    })
    
    expect(result.current.values.username).toBe('testuser')
  })

  it('should validate values on change', async () => {
    const { result } = renderHook(() => useForm({ initialValues, schema }))
    
    act(() => {
      result.current.handleChange('username', 'ab') // too short
    })
    
    // Validation is async in implementation
    await waitFor(() => {
      expect(result.current.errors.username).toBeDefined()
    })
    
    act(() => {
      result.current.handleChange('username', 'abc') // valid
    })
    
    await waitFor(() => {
      expect(result.current.errors.username).toBeUndefined()
    })
  })

  it('should handle form submission', async () => {
    const { result } = renderHook(() => useForm({ 
      initialValues: { username: 'testuser', email: 'test@example.com' }, 
      schema, 
      onSubmit: mockSubmit 
    }))
    
    act(() => {
      result.current.handleSubmit({ preventDefault: () => {} } as any)
    })
    
    await waitFor(() => {
      expect(result.current.isSubmitting).toBe(false)
    })
    
    expect(mockSubmit).toHaveBeenCalledWith({ username: 'testuser', email: 'test@example.com' })
  })

  it('should not submit if validation fails', async () => {
    const { result } = renderHook(() => useForm({ 
      initialValues: { username: 'a', email: 'not-email' }, 
      schema, 
      onSubmit: mockSubmit 
    }))
    
    act(() => {
      result.current.handleSubmit({ preventDefault: () => {} } as any)
    })
    
    await waitFor(() => {
      expect(result.current.errors.username).toBeDefined()
      expect(result.current.errors.email).toBeDefined()
    })
    
    expect(mockSubmit).not.toHaveBeenCalled()
  })

  it('should reset the form', () => {
    const { result } = renderHook(() => useForm({ initialValues, schema }))
    
    act(() => {
      result.current.handleChange('username', 'changed')
      result.current.reset()
    })
    
    expect(result.current.values).toEqual(initialValues)
  })
})
