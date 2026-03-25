import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/preact'
import { Select } from '../ui/Select'

describe('Select', () => {
  const options = [
    { value: 'en', label: 'English' },
    { value: 'ru', label: 'Русский' },
    { value: 'zh', label: '中文' },
  ]

  it('renders select with options', () => {
    render(<Select options={options} />)
    
    const select = screen.getByRole('combobox')
    expect(select).toBeInTheDocument()
    
    expect(screen.getByText('English')).toBeInTheDocument()
    expect(screen.getByText('Русский')).toBeInTheDocument()
    expect(screen.getByText('中文')).toBeInTheDocument()
  })

  it('handles selection change', () => {
    const handleChange = vi.fn()
    render(<Select options={options} onChange={handleChange} />)
    
    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'ru' } })
    
    expect(handleChange).toHaveBeenCalledTimes(1)
  })

  it('applies fullWidth prop', () => {
    render(<Select options={options} fullWidth />)
    
    const select = screen.getByRole('combobox')
    expect(select).toHaveClass('w-full')
  })

  it('renders with label', () => {
    render(<Select label="Language" options={options} />)
    
    const label = screen.getByText(/language/i)
    expect(label).toBeInTheDocument()
  })

  it('applies disabled state', () => {
    render(<Select options={options} disabled />)
    
    const select = screen.getByRole('combobox')
    expect(select).toBeDisabled()
  })

  it('renders with default value', () => {
    render(<Select options={options} value="ru" />)
    
    const select = screen.getByRole('combobox') as HTMLSelectElement
    expect(select.value).toBe('ru')
  })

  it('renders with error', () => {
    render(<Select options={options} error="Please select an option" />)
    
    const error = screen.getByText(/please select an option/i)
    expect(error).toBeInTheDocument()
  })
})
