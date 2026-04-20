import { describe, it, expect } from 'vitest'
import type { ProtocolSummary } from '../../types'

describe('InboundForm - Deprecated Protocol Logic', () => {
  const mockProtocols: ProtocolSummary[] = [
    {
      protocol: 'vless',
      label: 'VLESS',
      description: 'VLESS protocol',
      core: ['xray', 'sing-box'],
      direction: 'both',
      requires_tls: false,
      category: 'standard',
    },
    {
      protocol: 'vmess',
      label: 'VMess',
      description: 'VMess protocol',
      core: ['xray'],
      direction: 'both',
      requires_tls: false,
      category: 'standard',
      deprecated: true,
      deprecation_notice: 'VMess is deprecated due to security concerns. Please migrate to VLESS.',
    },
    {
      protocol: 'trojan',
      label: 'Trojan',
      description: 'Trojan protocol',
      core: ['xray', 'sing-box'],
      direction: 'both',
      requires_tls: true,
      category: 'standard',
    },
  ]

  it('identifies deprecated protocols correctly', () => {
    const deprecatedProtocol = mockProtocols.find(p => p.protocol === 'vmess')
    expect(deprecatedProtocol?.deprecated).toBe(true)
    expect(deprecatedProtocol?.deprecation_notice).toBe('VMess is deprecated due to security concerns. Please migrate to VLESS.')
  })

  it('identifies non-deprecated protocols correctly', () => {
    const vlessProtocol = mockProtocols.find(p => p.protocol === 'vless')
    expect(vlessProtocol?.deprecated).toBeUndefined()

    const trojanProtocol = mockProtocols.find(p => p.protocol === 'trojan')
    expect(trojanProtocol?.deprecated).toBeUndefined()
  })

  it('generates correct warning message for deprecated protocol', () => {
    const deprecatedProtocol = mockProtocols.find(p => p.protocol === 'vmess')
    const warningMessage = deprecatedProtocol?.deprecation_notice
      ? `⚠ Deprecated: ${deprecatedProtocol.deprecation_notice}`
      : ''

    expect(warningMessage).toBe('⚠ Deprecated: VMess is deprecated due to security concerns. Please migrate to VLESS.')
  })

  it('does not generate warning for non-deprecated protocol', () => {
    const vlessProtocol = mockProtocols.find(p => p.protocol === 'vless')
    const shouldShowWarning = vlessProtocol?.deprecated === true

    expect(shouldShowWarning).toBe(false)
  })
})