import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/preact'
import { Card } from '../ui/Card'

describe('Card', () => {
  it('renders card with children', () => {
    render(<Card>Card content</Card>)
    
    const card = screen.getByText(/card content/i)
    expect(card).toBeInTheDocument()
  })

  it('applies default padding', () => {
    render(<Card>Content</Card>)
    
    const card = screen.getByText(/content/i).closest('div')
    expect(card).toHaveClass('rounded-xl border')
  })

  it('applies custom className', () => {
    render(<Card className="custom-class">Content</Card>)
    
    const card = screen.getByText(/content/i)
    expect(card).toHaveClass('custom-class')
  })

  it('renders with header', () => {
    render(
      <Card>
        <h3>Header</h3>
        <p>Content</p>
      </Card>
    )
    
    const header = screen.getByRole('heading', { name: /header/i })
    const content = screen.getByText(/content/i)
    
    expect(header).toBeInTheDocument()
    expect(content).toBeInTheDocument()
  })

  it('applies default padding', () => {
    render(<Card>Content</Card>)

    const card = screen.getByText(/content/i).closest('div')
    expect(card).toHaveClass('rounded-xl border')
  })


})
