import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/preact'
import { Input } from '../ui/Input'

describe('Input', () => {
  it('renders input with label', () => {
    render(<Input label="Username" name="username" />)
    
    const label = screen.getByText(/username/i)
    const input = screen.getByRole('textbox')
    
    expect(label).toBeInTheDocument()
    expect(input).toBeInTheDocument()
  })

  it('handles text input', () => {
    const handleChange = vi.fn()
    render(<Input name="username" onChange={handleChange} />)
    
    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: 'test' } })
    
    expect(handleChange).toHaveBeenCalledTimes(1)
  })

  it('applies error state', () => {
    render(<Input label="Email" error="Invalid email" />)
    
    const error = screen.getByText(/invalid email/i)
    expect(error).toBeInTheDocument()
  })

  it('applies fullWidth prop', () => {
    render(<Input fullWidth />)
    
    const input = screen.getByRole('textbox')
    expect(input).toHaveClass('w-full')
  })

  it('renders password input', () => {
    render(<Input type="password" name="password" />)
    
    const input = screen.getByLabelText(/password/i) as HTMLInputElement
    expect(input.type).toBe('password')
  })

  it('applies disabled state', () => {
    render(<Input disabled />)
    
    const input = screen.getByRole('textbox')
    expect(input).toBeDisabled()
  })

  it('renders placeholder', () => {
    render(<Input placeholder="Enter your name" />)
    
    const input = screen.getByPlaceholderText(/enter your name/i)
    expect(input).toBeInTheDocument()
  })

  it('renders with value', () => {
    render(<Input value="test value" />)
    
    const input = screen.getByRole('textbox') as HTMLInputElement
    expect(input.value).toBe('test value')
  })
})
