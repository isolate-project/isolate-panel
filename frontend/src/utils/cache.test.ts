import { describe, it, expect, beforeEach, vi } from 'vitest'
import { cache } from './cache'

describe('Cache Utility', () => {
  beforeEach(() => {
    cache.clear()
  })

  it('should set and get values', () => {
    cache.set('test-key', { foo: 'bar' })
    expect(cache.get('test-key')).toEqual({ foo: 'bar' })
  })

  it('should return null for non-existent keys', () => {
    expect(cache.get('non-existent')).toBeNull()
  })

  it('should handle TTL and expire entries', () => {
    vi.useFakeTimers()
    const now = Date.now()
    vi.setSystemTime(now)

    cache.set('expiry-test', 'content', 1000) // 1 second TTL
    expect(cache.get('expiry-test')).toBe('content')

    vi.setSystemTime(now + 1001)
    expect(cache.get('expiry-test')).toBeNull()

    vi.useRealTimers()
  })

  it('should invalidate specific keys', () => {
    cache.set('k1', 'v1')
    cache.set('k2', 'v2')
    cache.invalidate('k1')
    
    expect(cache.get('k1')).toBeNull()
    expect(cache.get('k2')).toBe('v2')
  })

  it('should invalidate keys by pattern', () => {
    cache.set('users-1', 'u1')
    cache.set('users-2', 'u2')
    cache.set('settings', 's1')
    
    cache.invalidatePattern(/^users-/)
    
    expect(cache.get('users-1')).toBeNull()
    expect(cache.get('users-2')).toBeNull()
    expect(cache.get('settings')).toBe('s1')
  })

  it('should clear all entries', () => {
    cache.set('a', '1')
    cache.set('b', '2')
    cache.clear()
    
    expect(cache.get('a')).toBeNull()
    expect(cache.get('b')).toBeNull()
  })
})
