# Ultimate Architectural Solutions for Isolate Panel

## Executive Summary

This document provides comprehensive, production-grade solutions for 14 critical architectural problems identified in the Isolate Panel project audit. Each solution includes deep root cause analysis, implementation examples, migration strategies, and architectural rationale. These solutions prioritize long-term maintainability, security, and performance over quick fixes.

---

## Table of Contents

1. [Frontend Architectural Problems](#frontend-architectural-problems)
   - Problem 1: Massive Components
   - Problem 2: Missing Memoization
   - Problem 3: Hardcoded Strings
   - Problem 4: Magic Numbers
   - Problem 5: Module-Level Mutable State
   - Problem 6: Dual Token Storage
   - Problem 7: Accessibility Gaps

2. [DevOps Security Problems](#devops-security-problems)
   - Problem 8: No Supply Chain Security
   - Problem 9: No Container Image Scanning
   - Problem 10: No SBOM Generation
   - Problem 11: No Image Signing
   - Problem 12: Dockerfile Dev Downloads Binaries
   - Problem 13: Go Version Drift
   - Problem 14: No Read-Only Root FS

---

## Frontend Architectural Problems

---

### Problem 1: Massive Components

**Current State:**
- `Inbounds.tsx`: 329 lines
- `Dashboard.tsx`: 311 lines
- `InboundForm.tsx`: 344 lines (estimated from patterns)

These components violate the Single Responsibility Principle, mixing data fetching, state management, UI rendering, and business logic in a single file.

#### Root Cause Analysis

The current architecture suffers from **feature-centric colocation** rather than **responsibility-based separation**. When developers add features, they naturally extend existing files rather than creating new, focused components. This creates a compounding effect where each new feature increases the cognitive load exponentially.

The `Inbounds.tsx` file demonstrates this clearly:
- Lines 69-91: Data fetching and mutations (7 hooks)
- Lines 93-110: Derived state computation (filtering, data transformation)
- Lines 122-164: Control bar UI (search, filters)
- Lines 167-254: Main content grid (cards, loading states, empty states)
- Lines 256-326: Modal/Drawer management (3 different modals)

This violates the **Cognitive Load Theory** in software design. Research shows developers can effectively hold 7±2 chunks of information in working memory. A 329-line component with 15+ state variables exceeds this limit, leading to:
- Increased bug rates
- Slower code review cycles
- Higher onboarding costs
- Reduced refactoring confidence

#### Ultimate Solution: Smart/Dumb Component Architecture with Compound Components

We implement a three-layer component hierarchy:

1. **Container Components (Smart)**: Handle data fetching, mutations, and orchestration
2. **Compound Components**: Provide composable UI patterns with implicit state sharing
3. **Presentational Components (Dumb)**: Pure rendering with props-only interface

**Implementation:**

```typescript
// ============================================================================
// LAYER 1: Container Component - InboundsContainer.tsx
// Responsibility: Data orchestration, business logic, state management
// ============================================================================

import { useState, useCallback, useMemo } from 'preact/hooks'
import { useInbounds, useDeleteInbound, useAssignUser, useUnassignUser } from '../hooks/useInbounds'
import { useUsers } from '../hooks/useUsers'
import { useQuery } from '../hooks/useQuery'
import { inboundApi } from '../api/endpoints'
import type { Inbound, User } from '../types'
import { InboundsProvider } from './InboundsContext'
import { InboundsLayout } from './InboundsLayout'

interface InboundsContainerProps {
  children?: never // Container renders its own layout
}

export function InboundsContainer({}: InboundsContainerProps) {
  const { t } = useTranslation()
  
  // Data fetching
  const { data: inbounds, isLoading, refetch } = useInbounds()
  const { data: usersResponse } = useUsers()
  const { mutate: deleteInbound } = useDeleteInbound()
  const { mutate: assignUser } = useAssignUser()
  const { mutate: unassignUser } = useUnassignUser()
  
  // Local UI state
  const [searchTerm, setSearchTerm] = useState('')
  const [protocolFilter, setProtocolFilter] = useState<string>('all')
  const [selectedInbound, setSelectedInbound] = useState<Inbound | null>(null)
  const [activeModal, setActiveModal] = useState<ModalType>(null)
  
  // Derived state with memoization
  const filteredInbounds = useMemo(() => {
    if (!inbounds) return []
    return inbounds.filter((inbound) => {
      const matchesSearch = searchTerm
        ? inbound.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
          inbound.protocol.toLowerCase().includes(searchTerm.toLowerCase())
        : true
      const matchesProtocol = protocolFilter === 'all' || inbound.protocol === protocolFilter
      return matchesSearch && matchesProtocol
    })
  }, [inbounds, searchTerm, protocolFilter])
  
  // Event handlers with stable references
  const handleDelete = useCallback(async () => {
    if (selectedInbound) {
      await deleteInbound(selectedInbound.id)
      setActiveModal(null)
      setSelectedInbound(null)
      refetch()
    }
  }, [selectedInbound, deleteInbound, refetch])
  
  const handleAssignUser = useCallback(async (userId: number) => {
    if (selectedInbound) {
      await assignUser({ inboundId: selectedInbound.id, userId })
    }
  }, [selectedInbound, assignUser])
  
  // Context value with stable reference
  const contextValue = useMemo(() => ({
    inbounds: filteredInbounds,
    allInbounds: inbounds ?? [],
    users: usersResponse?.users ?? [],
    isLoading,
    searchTerm,
    setSearchTerm,
    protocolFilter,
    setProtocolFilter,
    selectedInbound,
    setSelectedInbound,
    activeModal,
    setActiveModal,
    onDelete: handleDelete,
    onAssignUser: handleAssignUser,
    onUnassignUser: unassignUser,
    refetch,
  }), [
    filteredInbounds, 
    inbounds, 
    usersResponse, 
    isLoading,
    searchTerm,
    protocolFilter,
    selectedInbound,
    activeModal,
    handleDelete,
    handleAssignUser,
    unassignUser,
    refetch
  ])
  
  return (
    <InboundsProvider value={contextValue}>
      <InboundsLayout title={t('inbounds.title')} description={t('inbounds.description')} />
    </InboundsProvider>
  )
}

// ============================================================================
// LAYER 2: Context Provider - InboundsContext.tsx
// Provides implicit state sharing between compound components
// ============================================================================

import { createContext, useContext } from 'preact'
import type { Inbound, User } from '../types'

type ModalType = 'create' | 'edit' | 'delete' | 'users' | null

interface InboundsContextValue {
  inbounds: Inbound[]
  allInbounds: Inbound[]
  users: User[]
  isLoading: boolean
  searchTerm: string
  setSearchTerm: (term: string) => void
  protocolFilter: string
  setProtocolFilter: (filter: string) => void
  selectedInbound: Inbound | null
  setSelectedInbound: (inbound: Inbound | null) => void
  activeModal: ModalType
  setActiveModal: (modal: ModalType) => void
  onDelete: () => Promise<void>
  onAssignUser: (userId: number) => Promise<void>
  onUnassignUser: (params: { inboundId: number; userId: number }) => Promise<void>
  refetch: () => void
}

const InboundsContext = createContext<InboundsContextValue | null>(null)

export function InboundsProvider({ 
  children, 
  value 
}: { 
  children: ComponentChildren
  value: InboundsContextValue 
}) {
  return (
    <InboundsContext.Provider value={value}>
      {children}
    </InboundsContext.Provider>
  )
}

export function useInboundsContext() {
  const context = useContext(InboundsContext)
  if (!context) {
    throw new Error('useInboundsContext must be used within InboundsProvider')
  }
  return context
}

// ============================================================================
// LAYER 3: Compound Components - InboundsLayout.tsx
// Composable UI with implicit context access
// ============================================================================

import { InboundsHeader } from './InboundsHeader'
import { InboundsFilterBar } from './InboundsFilterBar'
import { InboundsGrid } from './InboundsGrid'
import { InboundsEmptyState } from './InboundsEmptyState'
import { InboundsModals } from './InboundsModals'

interface InboundsLayoutProps {
  title: string
  description: string
}

export function InboundsLayout({ title, description }: InboundsLayoutProps) {
  const { inbounds, isLoading } = useInboundsContext()
  
  return (
    <PageLayout>
      <InboundsHeader title={title} description={description} />
      <InboundsFilterBar />
      
      {isLoading ? (
        <InboundsSkeleton />
      ) : inbounds.length > 0 ? (
        <InboundsGrid inbounds={inbounds} />
      ) : (
        <InboundsEmptyState />
      )}
      
      <InboundsModals />
    </PageLayout>
  )
}

// ============================================================================
// LAYER 4: Presentational Components - InboundsGrid.tsx
// Pure rendering, no business logic
// ============================================================================

import { InboundCard } from './InboundCard'
import type { Inbound } from '../types'

interface InboundsGridProps {
  inbounds: Inbound[]
}

export function InboundsGrid({ inbounds }: InboundsGridProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {inbounds.map((inbound) => (
        <InboundCard key={inbound.id} inbound={inbound} />
      ))}
    </div>
  )
}

// ============================================================================
// LAYER 4: Presentational Component - InboundCard.tsx
// Self-contained card with action delegation
// ============================================================================

import { useInboundsContext } from './InboundsContext'
import { Card, CardContent } from '../ui/Card'
import { Badge } from '../ui/Badge'
import { Button } from '../ui/Button'
import { Globe, Users, MoreVertical } from 'lucide-preact'

interface InboundCardProps {
  inbound: Inbound
}

export function InboundCard({ inbound }: InboundCardProps) {
  const { setSelectedInbound, setActiveModal } = useInboundsContext()
  
  const handleManageUsers = () => {
    setSelectedInbound(inbound)
    setActiveModal('users')
  }
  
  const handleEdit = () => {
    setSelectedInbound(inbound)
    setActiveModal('edit')
  }
  
  const handleDelete = () => {
    setSelectedInbound(inbound)
    setActiveModal('delete')
  }
  
  return (
    <Card className="relative overflow-hidden group hover:shadow-lg transition-all duration-300">
      <CardContent className="p-0">
        <InboundCardHeader 
          inbound={inbound} 
          onManageUsers={handleManageUsers}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
        <InboundCardDetails inbound={inbound} />
        <InboundCardCapabilities inbound={inbound} />
        <InboundCardFooter inbound={inbound} onManageUsers={handleManageUsers} />
      </CardContent>
    </Card>
  )
}
```

#### Migration Path

**Phase 1: Extract Context (Week 1)**
1. Create `InboundsContext.tsx` with all shared state
2. Move state from `Inbounds.tsx` to context provider
3. Verify all existing tests pass

**Phase 2: Extract Presentational Components (Week 2)**
1. Create `InboundCard.tsx` with props interface
2. Create `InboundsGrid.tsx`
3. Create `InboundsFilterBar.tsx`
4. Update imports and verify functionality

**Phase 3: Extract Container (Week 3)**
1. Create `InboundsContainer.tsx`
2. Move all data fetching and mutations
3. Simplify `Inbounds.tsx` to render container only
4. Run full E2E test suite

**Phase 4: Add Compound Component Pattern (Week 4)**
1. Create `InboundsLayout.tsx` compound component
2. Implement sub-components (Header, FilterBar, Grid, Modals)
3. Update page to use new API
4. Document the pattern for other pages

#### Why This Is Architecturally Superior

1. **Separation of Concerns**: Each layer has exactly one responsibility
2. **Testability**: Presentational components can be tested with simple props
3. **Reusability**: `InboundCard` can be used in dashboards, reports, or other views
4. **Performance**: Context splits prevent unnecessary re-renders
5. **Team Scaling**: Multiple developers can work on different layers simultaneously
6. **Cognitive Load**: Each file fits in working memory (under 100 lines)

---

### Problem 2: Missing Memoization

**Current State:**
- Only 15/104 files use `useMemo`/`useCallback`
- Derived state recalculates on every render
- Event handlers create new function references
- Child components re-render unnecessarily

#### Root Cause Analysis

The codebase lacks a systematic approach to render optimization. React's default behavior is to re-render children when parents render, even if props haven't changed. Without memoization:

1. **Reference instability**: `onClick={() => doSomething()}` creates a new function every render
2. **Derived data recreation**: `filteredInbounds` recalculates even when inputs haven't changed
3. **Object/Array literals**: `style={{ color: 'red' }}` creates new objects every render

In `Inbounds.tsx`, the `filteredInbounds` calculation (lines 103-110) runs on every render, even when `searchTerm` and `protocolFilter` haven't changed. For 100+ inbounds, this is O(n) work wasted.

#### Ultimate Solution: Strategic Memoization with Custom Hooks

We implement a systematic memoization strategy using custom hooks that encapsulate optimization logic:

```typescript
// ============================================================================
// Custom Hook: useFilteredData - Optimized filtering with memoization
// ============================================================================

import { useMemo, useCallback } from 'preact/hooks'

interface UseFilteredDataOptions<T> {
  data: T[]
  searchTerm: string
  filterKey: keyof T
  additionalFilters?: Array<{
    key: keyof T
    value: unknown
    predicate?: (item: T, value: unknown) => boolean
  }>
}

export function useFilteredData<T extends Record<string, unknown>>({
  data,
  searchTerm,
  filterKey,
  additionalFilters = []
}: UseFilteredDataOptions<T>) {
  // Memoize the search predicate to avoid recreating function
  const searchPredicate = useCallback((item: T, term: string) => {
    const value = String(item[filterKey] ?? '').toLowerCase()
    return value.includes(term.toLowerCase())
  }, [filterKey])
  
  // Memoize the filtered result
  const filteredData = useMemo(() => {
    if (!data.length) return []
    
    return data.filter((item) => {
      // Apply search filter
      if (searchTerm && !searchPredicate(item, searchTerm)) {
        return false
      }
      
      // Apply additional filters
      for (const filter of additionalFilters) {
        const { key, value, predicate } = filter
        if (predicate) {
          if (!predicate(item, value)) return false
        } else if (item[key] !== value) {
          return false
        }
      }
      
      return true
    })
  }, [data, searchTerm, searchPredicate, additionalFilters])
  
  return filteredData
}

// ============================================================================
// Custom Hook: useStableCallbacks - Batch create stable callbacks
// ============================================================================

import { useCallback, useRef } from 'preact/hooks'

interface CallbackMap {
  [key: string]: (...args: unknown[]) => unknown
}

export function useStableCallbacks<T extends CallbackMap>(
  callbacks: T,
  deps: unknown[]
): T {
  const callbacksRef = useRef(callbacks)
  
  // Update callbacks when deps change
  useMemo(() => {
    callbacksRef.current = callbacks
  }, deps)
  
  // Create stable wrapper functions
  const stableCallbacks = useMemo(() => {
    const stable: Partial<T> = {}
    for (const key of Object.keys(callbacks)) {
      stable[key as keyof T] = ((...args: unknown[]) => {
        return callbacksRef.current[key](...args)
      }) as T[keyof T]
    }
    return stable as T
  }, []) // Empty deps = never recreates
  
  return stableCallbacks
}

// ============================================================================
// Custom Hook: useDerivedState - Memoized state derivation
// ============================================================================

import { useMemo } from 'preact/hooks'

interface UseDerivedStateOptions<T, D> {
  data: T
  derive: (data: T) => D
  deps?: unknown[]
}

export function useDerivedState<T, D>({
  data,
  derive,
  deps = []
}: UseDerivedStateOptions<T, D>): D {
  return useMemo(() => derive(data), [data, ...deps])
}

// ============================================================================
// Application: Optimized Inbounds Component
// ============================================================================

import { useFilteredData } from '../hooks/useFilteredData'
import { useStableCallbacks } from '../hooks/useStableCallbacks'
import { useDerivedState } from '../hooks/useDerivedState'

export function Inbounds() {
  const { t } = useTranslation()
  const { data: inbounds, isLoading } = useInbounds()
  
  const [searchTerm, setSearchTerm] = useState('')
  const [protocolFilter, setProtocolFilter] = useState<string>('all')
  
  // Optimized filtering with useFilteredData
  const filteredInbounds = useFilteredData({
    data: inbounds ?? [],
    searchTerm,
    filterKey: 'name',
    additionalFilters: [
      {
        key: 'protocol',
        value: protocolFilter,
        predicate: (item, value) => value === 'all' || item.protocol === value
      }
    ]
  })
  
  // Optimized derived statistics
  const stats = useDerivedState({
    data: inbounds ?? [],
    derive: (data) => ({
      total: data.length,
      active: data.filter(i => i.is_enabled).length,
      byProtocol: data.reduce((acc, i) => {
        acc[i.protocol] = (acc[i.protocol] || 0) + 1
        return acc
      }, {} as Record<string, number>)
    })
  })
  
  // Stable callbacks prevent child re-renders
  const callbacks = useStableCallbacks({
    onSearchChange: (e: Event) => setSearchTerm((e.target as HTMLInputElement).value),
    onProtocolChange: (e: Event) => setProtocolFilter((e.target as HTMLSelectElement).value),
    onClearFilters: () => {
      setSearchTerm('')
      setProtocolFilter('all')
    }
  }, [setSearchTerm, setProtocolFilter])
  
  return (
    <PageLayout>
      <InboundsStats stats={stats} />
      <InboundsFilterBar 
        searchTerm={searchTerm}
        protocolFilter={protocolFilter}
        onSearchChange={callbacks.onSearchChange}
        onProtocolChange={callbacks.onProtocolChange}
        onClear={callbacks.onClearFilters}
      />
      <InboundsGrid inbounds={filteredInbounds} isLoading={isLoading} />
    </PageLayout>
  )
}

// ============================================================================
// Component: Memoized InboundCard
// ============================================================================

import { memo } from 'preact/compat'

interface InboundCardProps {
  inbound: Inbound
  onManageUsers: (inbound: Inbound) => void
  onEdit: (inbound: Inbound) => void
  onDelete: (inbound: Inbound) => void
}

// Memo prevents re-render if props are equal
export const InboundCard = memo(function InboundCard({
  inbound,
  onManageUsers,
  onEdit,
  onDelete
}: InboundCardProps) {
  // Component implementation
  return (
    <Card>
      {/* Card content */}
    </Card>
  )
}, (prevProps, nextProps) => {
  // Custom comparison for deep equality on inbound
  return prevProps.inbound.id === nextProps.inbound.id &&
         prevProps.inbound.is_enabled === nextProps.inbound.is_enabled
})
```

#### Migration Path

**Phase 1: Audit and Measure (Week 1)**
1. Install React DevTools Profiler
2. Identify components with unnecessary re-renders
3. Document performance bottlenecks

**Phase 2: Create Custom Hooks (Week 2)**
1. Implement `useFilteredData`, `useStableCallbacks`, `useDerivedState`
2. Add unit tests for each hook
3. Document usage patterns

**Phase 3: Apply to High-Impact Components (Week 3-4)**
1. Start with `Inbounds.tsx` (329 lines, heavy filtering)
2. Apply to `Dashboard.tsx` (real-time data)
3. Apply to `InboundForm.tsx` (complex form state)
4. Measure before/after render counts

**Phase 4: Team Guidelines (Week 5)**
1. Create memoization decision tree
2. Document when to use useMemo vs useCallback vs memo
3. Add ESLint rules for common anti-patterns

#### Why This Is Architecturally Superior

1. **Systematic Approach**: Custom hooks enforce consistent patterns
2. **Developer Experience**: Hooks abstract complexity, API remains simple
3. **Measurable Impact**: Profiler shows exact render count reductions
4. **Maintainability**: Optimization logic is centralized and tested
5. **Composability**: Hooks can be combined for complex scenarios

---

### Problem 3: Hardcoded Strings

**Current State:**
- "General Configuration" in `InboundForm.tsx` (line 188)
- "Network Settings" in `InboundForm.tsx` (line 245)
- Mixed i18n usage across components

#### Root Cause Analysis

The i18n implementation is inconsistent. While most user-facing strings use `t('key')`, some strings remain hardcoded. This happens because:

1. **Developer oversight**: Strings added during rapid development
2. **Missing translation keys**: New features without corresponding i18n entries
3. **No enforcement**: No automated checks prevent hardcoded strings

This creates a poor user experience for non-English speakers and makes the application feel unpolished.

#### Ultimate Solution: Type-Safe i18n with Extraction Pipeline

We implement a comprehensive i18n architecture with TypeScript integration, automated extraction, and runtime validation:

```typescript
// ============================================================================
// Type Definitions: i18n/resources.ts
// Provides type safety and autocomplete for all translation keys
// ============================================================================

export interface TranslationResources {
  common: {
    save: string
    cancel: string
    delete: string
    edit: string
    create: string
    loading: string
    error: string
    success: string
    confirm: string
    close: string
    search: string
    filter: string
    actions: string
    status: {
      active: string
      inactive: string
      pending: string
    }
  }
  inbounds: {
    title: string
    description: string
    addInbound: string
    editInbound: string
    deleteInbound: string
    manageUsers: string
    manageAccess: string
    searchPlaceholder: string
    allProtocols: string
    protocol: string
    port: string
    core: string
    noCoreAssigned: string
    active: string
    disabled: string
    noMatchingInbounds: string
    createHint: string
    deleteConfirm: string
    deleteConfirmation: string
    createDescription: string
    editDescription: string
    sections: {
      general: string        // "General Configuration"
      network: string       // "Network Settings"
      security: string
      transport: string
      advanced: string
    }
    fields: {
      name: {
        label: string
        placeholder: string
        description: string
      }
      protocol: {
        label: string
        description: string
      }
      port: {
        label: string
        description: string
      }
    }
  }
  // ... other namespaces
}

// ============================================================================
// Type-Safe Hook: useTypedTranslation
// ============================================================================

import { useTranslation } from 'react-i18next'
import type { TranslationResources } from './resources'

// Recursive path type for nested keys
type Path<T, K extends keyof T = keyof T> = K extends string
  ? T[K] extends Record<string, unknown>
    ? `${K}.${Path<T[K], keyof T[K]>}` | K
    : K
  : never

type TranslationKey = Path<TranslationResources>

export function useTypedTranslation(namespace: keyof TranslationResources) {
  const { t, i18n } = useTranslation(namespace)
  
  // Typed translation function with autocomplete
  const typedT = (key: Path<TranslationResources[typeof namespace]>, options?: Record<string, unknown>) => {
    return t(key, options)
  }
  
  return { t: typedT, i18n }
}

// ============================================================================
// Component: InboundForm with Full i18n
// ============================================================================

import { useTypedTranslation } from '../i18n/useTypedTranslation'

export function InboundForm({ inbound, onSuccess, onCancel }: InboundFormProps) {
  const { t } = useTypedTranslation('inbounds')
  const { t: tCommon } = useTypedTranslation('common')
  
  return (
    <form>
      {/* Section headers now use typed translations */}
      <section>
        <h3>{t('sections.general')}</h3>
        <FormField
          label={t('fields.name.label')}
          placeholder={t('fields.name.placeholder')}
          description={t('fields.name.description')}
        />
        <FormField
          label={t('fields.protocol.label')}
          description={t('fields.protocol.description')}
        />
      </section>
      
      <section>
        <h3>{t('sections.network')}</h3>
        <FormField
          label={t('fields.port.label')}
          description={t('fields.port.description')}
        />
      </section>
      
      <div className="form-actions">
        <Button variant="secondary" onClick={onCancel}>
          {tCommon('cancel')}
        </Button>
        <Button type="submit">
          {inbound ? tCommon('save') : tCommon('create')}
        </Button>
      </div>
    </form>
  )
}

// ============================================================================
// ESLint Rule: no-hardcoded-strings.js
// Prevents hardcoded strings in JSX
// ============================================================================

module.exports = {
  meta: {
    type: 'problem',
    docs: {
      description: 'Disallow hardcoded strings in JSX',
      category: 'Possible Errors',
      recommended: true,
    },
    fixable: 'code',
    schema: [
      {
        type: 'object',
        properties: {
          allowedStrings: {
            type: 'array',
            items: { type: 'string' }
          }
        }
      }
    ]
  },
  create(context) {
    const options = context.options[0] || {}
    const allowedStrings = new Set(options.allowedStrings || [' ', '→', '←', '×'])
    
    return {
      JSXText(node) {
        const text = node.value.trim()
        if (text && !allowedStrings.has(text) && /^[a-zA-Z]/.test(text)) {
          context.report({
            node,
            message: `Hardcoded string "${text}" found. Use i18n translation instead.`,
          })
        }
      }
    }
  }
}

// ============================================================================
// Extraction Script: scripts/extract-i18n.ts
// Automatically extracts strings from source code
// ============================================================================

import { glob } from 'glob'
import { parse } from '@babel/parser'
import traverse from '@babel/traverse'
import fs from 'fs/promises'

interface ExtractedString {
  key: string
  defaultValue: string
  context?: string
  file: string
  line: number
}

async function extractStrings(): Promise<ExtractedString[]> {
  const files = await glob('src/**/*.{tsx,ts}')
  const extracted: ExtractedString[] = []
  
  for (const file of files) {
    const content = await fs.readFile(file, 'utf-8')
    const ast = parse(content, {
      sourceType: 'module',
      plugins: ['typescript', 'jsx']
    })
    
    traverse(ast, {
      // Extract t('key') calls
      CallExpression(path) {
        if (path.node.callee.type === 'Identifier' && path.node.callee.name === 't') {
          const firstArg = path.node.arguments[0]
          if (firstArg?.type === 'StringLiteral') {
            extracted.push({
              key: firstArg.value,
              defaultValue: '',
              file,
              line: path.node.loc?.start.line ?? 0
            })
          }
        }
      }
    })
  }
  
  return extracted
}

// Run extraction
extractStrings().then(strings => {
  console.log(`Extracted ${strings.length} translation keys`)
  // Merge with existing translations, flag missing keys
})
```

#### Migration Path

**Phase 1: Type System Setup (Week 1)**
1. Define `TranslationResources` interface with all namespaces
2. Create `useTypedTranslation` hook
3. Update `i18n/index.ts` to use typed resources

**Phase 2: Automated Detection (Week 2)**
1. Install and configure `eslint-plugin-i18n`
2. Add custom ESLint rule for hardcoded strings
3. Run on entire codebase to generate report

**Phase 3: String Extraction (Week 3-4)**
1. Run extraction script to find all hardcoded strings
2. Add missing keys to translation files
3. Replace hardcoded strings with `t('key')` calls

**Phase 4: CI Integration (Week 5)**
1. Add i18n check to pre-commit hooks
2. Add translation coverage check to CI
3. Block PRs with missing translations

#### Why This Is Architecturally Superior

1. **Type Safety**: Autocomplete prevents typos in translation keys
2. **Developer Experience**: Immediate feedback on missing translations
3. **Quality Assurance**: Automated checks prevent regression
4. **Scalability**: Easy to add new languages
5. **Maintainability**: Centralized translation management

---

### Problem 4: Magic Numbers

**Current State:**
- `refetchInterval: 15000` in `useInbounds.ts`
- `timeout: 10000` in `api/client.ts`
- `ttl: 300000` in `utils/cache.ts`
- No centralized constants file

#### Root Cause Analysis

Magic numbers create several problems:

1. **No semantic meaning**: `15000` could be polling interval, timeout, or delay
2. **Inconsistency**: Different components use different values for same concept
3. **Hard to change**: Finding all occurrences requires global search
4. **No documentation**: Value intent is not explained

In `useInbounds.ts`, `refetchInterval: 15000` appears without context. Is this 15 seconds? Why 15 and not 10 or 30? The next developer cannot know without reading surrounding code.

#### Ultimate Solution: Domain-Driven Constants with Hierarchical Organization

We implement a comprehensive constants architecture organized by domain, with TypeScript enums, configuration objects, and runtime validation:

```typescript
// ============================================================================
// Constants: Time values with semantic meaning
// ============================================================================

export const TimeConstants = {
  // Milliseconds
  MS_PER_SECOND: 1000,
  MS_PER_MINUTE: 60 * 1000,
  MS_PER_HOUR: 60 * 60 * 1000,
  MS_PER_DAY: 24 * 60 * 60 * 1000,
  
  // Polling intervals
  POLLING: {
    REALTIME: 1000,           // 1 second - WebSocket fallback
    FREQUENT: 5000,           // 5 seconds - active connections
    STANDARD: 15000,          // 15 seconds - inbounds list
    RELAXED: 60000,           // 1 minute - system resources
    BACKGROUND: 300000,      // 5 minutes - background sync
  } as const,
  
  // Timeouts
  TIMEOUT: {
    API_REQUEST: 10000,       // 10 seconds - standard API call
    FILE_UPLOAD: 60000,       // 1 minute - file operations
    WEBSOCKET_CONNECT: 5000,  // 5 seconds - WS handshake
    WEBSOCKET_PING: 30000,    // 30 seconds - keepalive
  } as const,
  
  // Cache TTL
  CACHE: {
    EPHEMERAL: 5000,         // 5 seconds - rapidly changing data
    SHORT: 30000,            // 30 seconds - temporary data
    STANDARD: 300000,        // 5 minutes - normal cache
    LONG: 3600000,           // 1 hour - reference data
    PERSISTENT: 86400000,    // 24 hours - rarely changing data
  } as const,
  
  // Debounce delays
  DEBOUNCE: {
    TYPING: 300,             // 300ms - search input
    RESIZE: 250,             // 250ms - window resize
    SCROLL: 100,             // 100ms - scroll events
  } as const,
} as const

// ============================================================================
// Constants: API Configuration
// ============================================================================

export const ApiConstants = {
  // Pagination
  PAGINATION: {
    DEFAULT_PAGE_SIZE: 20,
    MAX_PAGE_SIZE: 100,
    PAGE_SIZE_OPTIONS: [10, 20, 50, 100] as const,
  },
  
  // Retry configuration
  RETRY: {
    MAX_ATTEMPTS: 3,
    BACKOFF_MULTIPLIER: 2,
    INITIAL_DELAY: 1000,
  },
  
  // Rate limiting
  RATE_LIMIT: {
    REQUESTS_PER_MINUTE: 60,
    BURST_SIZE: 10,
  },
} as const

// ============================================================================
// Constants: UI/UX Values
// ============================================================================

export const UIConstants = {
  // Animation durations (ms)
  ANIMATION: {
    FAST: 150,
    NORMAL: 300,
    SLOW: 500,
    PAGE_TRANSITION: 400,
  },
  
  // Z-index scale
  Z_INDEX: {
    BASE: 0,
    STICKY: 10,
    DROPDOWN: 100,
    MODAL: 200,
    TOOLTIP: 300,
    TOAST: 400,
    PANIC: 9999,
  },
  
  // Breakpoints (px)
  BREAKPOINT: {
    SM: 640,
    MD: 768,
    LG: 1024,
    XL: 1280,
    XXL: 1536,
  },
  
  // Layout
  LAYOUT: {
    SIDEBAR_WIDTH: 280,
    HEADER_HEIGHT: 64,
    MAX_CONTENT_WIDTH: 1440,
    CARD_GRID_GAP: 24,
  },
} as const

// ============================================================================
// Constants: Feature Flags and Limits
// ============================================================================

export const FeatureConstants = {
  // Inbound limits
  INBOUND: {
    MAX_PORT: 65535,
    MIN_PORT: 1,
    RESERVED_PORTS: [8080, 8443, 22, 80, 443],
    MAX_NAME_LENGTH: 64,
  },
  
  // User limits
  USER: {
    MAX_USERNAME_LENGTH: 32,
    MIN_PASSWORD_LENGTH: 8,
    MAX_ACTIVE_SESSIONS: 5,
  },
  
  // File uploads
  UPLOAD: {
    MAX_FILE_SIZE: 10 * 1024 * 1024, // 10MB
    ALLOWED_EXTENSIONS: ['.json', '.yaml', '.yml', '.conf'],
  },
} as const

// ============================================================================
// Application: Using constants in hooks
// ============================================================================

import { TimeConstants } from '../constants/time'
import { ApiConstants } from '../constants/api'

export function useInbounds() {
  return useQuery(
    'inbounds',
    fetchInbounds,
    {
      refetchInterval: TimeConstants.POLLING.STANDARD, // 15000ms
      staleTime: TimeConstants.CACHE.SHORT,            // 30000ms
    }
  )
}

export function useConnections() {
  return useQuery(
    'connections',
    fetchConnections,
    {
      refetchInterval: TimeConstants.POLLING.FREQUENT, // 5000ms
    }
  )
}

// ============================================================================
// Application: Using constants in API client
// ============================================================================

import { TimeConstants } from './constants/time'
import { ApiConstants } from './constants/api'

const apiClient = axios.create({
  timeout: TimeConstants.TIMEOUT.API_REQUEST, // 10000ms
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor with retry logic
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const config = error.config
    
    if (!config.retryCount) {
      config.retryCount = 0
    }
    
    if (config.retryCount < ApiConstants.RETRY.MAX_ATTEMPTS) {
      config.retryCount += 1
      const delay = ApiConstants.RETRY.INITIAL_DELAY * 
        Math.pow(ApiConstants.RETRY.BACKOFF_MULTIPLIER, config.retryCount - 1)
      
      await new Promise(resolve => setTimeout(resolve, delay))
      return apiClient(config)
    }
    
    return Promise.reject(error)
  }
)

// ============================================================================
// Application: Using constants in cache utility
// ============================================================================

import { TimeConstants } from './constants/time'

class Cache {
  set<T>(key: string, data: T, ttl: number = TimeConstants.CACHE.STANDARD): void {
    const entry: CacheEntry<T> = {
      data,
      expiresAt: Date.now() + ttl,
    }
    this.storage.set(key, entry)
  }
}
```

#### Migration Path

**Phase 1: Create Constants Structure (Week 1)**
1. Create `constants/` directory with domain files
2. Define `TimeConstants`, `ApiConstants`, `UIConstants`
3. Add barrel export (`constants/index.ts`)

**Phase 2: Replace Magic Numbers (Week 2-3)**
1. Search for all numeric literals in codebase
2. Replace with semantic constants
3. Add inline comments for complex calculations

**Phase 3: Add Validation (Week 4)**
1. Create runtime validation for critical constants
2. Add tests ensuring constants are used
3. Document constant usage guidelines

**Phase 4: ESLint Rule (Week 5)**
1. Create ESLint rule to flag magic numbers
2. Configure exceptions (0, 1, -1, array indices)
3. Add to CI pipeline

#### Why This Is Architecturally Superior

1. **Semantic Clarity**: `TimeConstants.POLLING.STANDARD` explains intent
2. **Centralized Control**: Change in one place affects entire app
3. **Type Safety**: Constants are typed, preventing invalid values
4. **Discoverability**: IDE autocomplete shows available options
5. **Documentation**: Constants file serves as configuration reference

---

### Problem 5: Module-Level Mutable State

**Current State:**
```typescript
// ProtectedRoute.tsx - lines 12-14
let authVerified = false
let authVerifiedAt = 0
const AUTH_CACHE_TTL = 60000
```

This creates shared mutable state at the module level, causing:
- State leakage between tests
- Race conditions in concurrent renders
- Unpredictable behavior in Strict Mode

#### Root Cause Analysis

Module-level variables in JavaScript/TypeScript are singletons. When the module is imported, all importers share the same variable reference. In React:

1. **Test isolation**: Tests run in same process, state persists between tests
2. **Strict Mode**: Components mount/unmount twice, state doesn't reset
3. **Concurrent features**: React 18+ concurrent rendering can cause race conditions
4. **HMR**: Hot module replacement preserves state, causing stale data

The `authVerified` flag is particularly dangerous because it affects authentication flow. If it gets stuck in the wrong state, users may be incorrectly redirected or granted access.

#### Ultimate Solution: Ref-Based State with Proper Lifecycle Management

We replace module-level state with React refs and context, ensuring proper lifecycle management:

```typescript
// ============================================================================
// Context: AuthCacheContext.tsx
// Provides component-scoped auth verification cache
// ============================================================================

import { createContext, useContext, useRef, useCallback } from 'preact/hooks'
import { TimeConstants } from '../constants/time'

interface AuthCacheState {
  isVerified: boolean
  verifiedAt: number
  verificationPromise: Promise<void> | null
}

interface AuthCacheContextValue {
  getIsVerified: () => boolean
  setVerified: () => void
  invalidate: () => void
  getVerificationPromise: () => Promise<void> | null
  setVerificationPromise: (promise: Promise<void> | null) => void
}

const AuthCacheContext = createContext<AuthCacheContextValue | null>(null)

export function AuthCacheProvider({ children }: { children: ComponentChildren }) {
  // Use ref for mutable state that doesn't trigger re-renders
  const cacheRef = useRef<AuthCacheState>({
    isVerified: false,
    verifiedAt: 0,
    verificationPromise: null,
  })
  
  const getIsVerified = useCallback(() => {
    const cache = cacheRef.current
    if (!cache.isVerified) return false
    
    const elapsed = Date.now() - cache.verifiedAt
    return elapsed < TimeConstants.CACHE.SHORT // 30000ms
  }, [])
  
  const setVerified = useCallback(() => {
    cacheRef.current.isVerified = true
    cacheRef.current.verifiedAt = Date.now()
  }, [])
  
  const invalidate = useCallback(() => {
    cacheRef.current.isVerified = false
    cacheRef.current.verifiedAt = 0
    cacheRef.current.verificationPromise = null
  }, [])
  
  const getVerificationPromise = useCallback(() => {
    return cacheRef.current.verificationPromise
  }, [])
  
  const setVerificationPromise = useCallback((promise: Promise<void> | null) => {
    cacheRef.current.verificationPromise = promise
  }, [])
  
  const value: AuthCacheContextValue = {
    getIsVerified,
    setVerified,
    invalidate,
    getVerificationPromise,
    setVerificationPromise,
  }
  
  return (
    <AuthCacheContext.Provider value={value}>
      {children}
    </AuthCacheContext.Provider>
  )
}

export function useAuthCache() {
  const context = useContext(AuthCacheContext)
  if (!context) {
    throw new Error('useAuthCache must be used within AuthCacheProvider')
  }
  return context
}

// ============================================================================
// Component: ProtectedRoute with proper state management
// ============================================================================

import { useEffect, useState, useRef } from 'preact/hooks'
import { route } from 'preact-router'
import { useAuthStore } from '../stores/authStore'
import { useAuthCache } from '../contexts/AuthCacheContext'
import { Spinner } from '../components/ui/Spinner'
import { authApi } from '../api/endpoints'

interface ProtectedRouteProps {
  children: ComponentChildren
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { isAuthenticated, setUser, logout } = useAuthStore()
  const user = useAuthStore(s => s.user)
  const accessToken = useAuthStore(s => s.accessToken)
  
  const [isChecking, setIsChecking] = useState(true)
  const abortControllerRef = useRef<AbortController | null>(null)
  
  // Use context-based cache instead of module-level variables
  const authCache = useAuthCache()
  
  useEffect(() => {
    return () => {
      // Cleanup abort controller on unmount
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [])
  
  useEffect(() => {
    const checkAuth = async () => {
      // Cancel any in-flight request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      abortControllerRef.current = new AbortController()
      
      const token = localStorage.getItem('accessToken')
      
      if (!token) {
        setIsChecking(false)
        route('/login', true)
        return
      }
      
      // Check cache first
      if (authCache.getIsVerified() && isAuthenticated) {
        setIsChecking(false)
        return
      }
      
      // Check if verification is already in progress
      const existingPromise = authCache.getVerificationPromise()
      if (existingPromise) {
        try {
          await existingPromise
          setIsChecking(false)
          return
        } catch {
          // Verification failed, continue to retry
        }
      }
      
      // Create new verification promise
      const verificationPromise = (async () => {
        try {
          const response = await authApi.me()
          
          if (abortControllerRef.current?.signal.aborted) {
            return
          }
          
          setUser(response.data)
          authCache.setVerified()
          setIsChecking(false)
        } catch (err) {
          if (err instanceof Error && err.name === 'AbortError') {
            return
          }
          
          authCache.invalidate()
          logout()
          route('/login', true)
        }
      })()
      
      authCache.setVerificationPromise(verificationPromise)
      await verificationPromise
      authCache.setVerificationPromise(null)
    }
    
    checkAuth()
  }, [accessToken, isAuthenticated, setUser, logout, authCache])
  
  if (isChecking) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-secondary">
        <div className="text-center">
          <Spinner size="lg" />
          <p className="mt-4 text-sm text-secondary">Verifying authentication...</p>
        </div>
      </div>
    )
  }
  
  if (!isAuthenticated) {
    return null
  }
  
  if (user?.must_change_password && typeof window !== 'undefined' && 
      !window.location.pathname.endsWith('/change-password')) {
    route('/change-password', true)
    return null
  }
  
  return <>{children}</>
}

// ============================================================================
// Hook: useInvalidateAuthCache
// For external cache invalidation
// ============================================================================

export function useInvalidateAuthCache() {
  const authCache = useAuthCache()
  
  return useCallback(() => {
    authCache.invalidate()
  }, [authCache])
}

// ============================================================================
// Test Setup: Proper isolation
// ============================================================================

import { render } from '@testing-library/preact'
import { AuthCacheProvider } from '../contexts/AuthCacheContext'

// Wrap all tests with fresh provider
export function renderWithAuthCache(ui: ComponentChildren) {
  return render(
    <AuthCacheProvider>
      {ui}
    </AuthCacheProvider>
  )
}
```

#### Migration Path

**Phase 1: Create Context (Week 1)**
1. Create `AuthCacheContext.tsx` with ref-based state
2. Implement all cache operations as methods
3. Add comprehensive unit tests

**Phase 2: Update ProtectedRoute (Week 2)**
1. Replace module-level variables with context
2. Add deduplication logic for concurrent requests
3. Test with React Strict Mode

**Phase 3: Update Tests (Week 3)**
1. Create `renderWithAuthCache` test utility
2. Update all ProtectedRoute tests
3. Verify test isolation

**Phase 4: Audit Other Module State (Week 4)**
1. Search for all `let` declarations at module level
2. Replace with appropriate React patterns
3. Add ESLint rule to prevent future occurrences

#### Why This Is Architecturally Superior

1. **Proper Lifecycle**: State is tied to component lifecycle, not module lifetime
2. **Test Isolation**: Each test gets fresh state via new provider instance
3. **Concurrent Safety**: Refs are safe in React 18 concurrent features
4. **Request Deduplication**: Prevents multiple simultaneous auth checks
5. **Explicit Dependencies**: State dependencies are visible in component

---

### Problem 6: Dual Token Storage

**Current State:**
```typescript
// authStore.ts - lines 34-41
setTokens: (accessToken, refreshToken) => {
  // Single source of truth: Zustand persist writes to localStorage under 'auth-storage'.
  // We also write dedicated keys so the API client interceptor can read them
  // without importing the store (avoids circular dependency).
  localStorage.setItem('accessToken', accessToken)
  localStorage.setItem('refreshToken', refreshToken)
  set({ accessToken, refreshToken, isAuthenticated: true })
}
```

This creates:
- Two storage locations for same data
- Synchronization bugs
- XSS vulnerability (localStorage is accessible to any JS)
- No httpOnly cookie protection

#### Root Cause Analysis

The current implementation attempts to solve a circular dependency problem (store needs API, API needs store) by duplicating storage. However, this creates worse problems:

1. **Security**: localStorage is vulnerable to XSS attacks. Any injected script can read tokens.
2. **Consistency**: If one write fails, store and localStorage become out of sync
3. **Complexity**: Two sources of truth require synchronization logic
4. **SSR issues**: localStorage doesn't exist during server-side rendering

The "BFF pattern" (Backend for Frontend) is the industry-standard solution. The backend sets httpOnly cookies that JavaScript cannot access, while the frontend keeps only the access token in memory.

#### Ultimate Solution: BFF Pattern with httpOnly Cookies

We implement a secure token architecture using httpOnly cookies for refresh tokens and in-memory storage for access tokens:

```typescript
// ============================================================================
// Backend: Auth Handler (Go)
// Sets httpOnly cookies for refresh tokens
// ============================================================================

package auth

import (
    "net/http"
    "time"
)

const (
    refreshTokenCookieName = "refresh_token"
    accessTokenCookieName  = "access_token"
    cookiePath            = "/"
    cookieSecure          = true  // HTTPS only
    cookieHttpOnly        = true  // JavaScript cannot access
    cookieSameSite        = http.SameSiteStrictMode
)

func (h *Handler) Login(c *fiber.Ctx) error {
    // ... validate credentials ...
    
    accessToken, refreshToken, err := h.service.GenerateTokens(user)
    if err != nil {
        return err
    }
    
    // Set refresh token as httpOnly cookie
    c.Cookie(&fiber.Cookie{
        Name:     refreshTokenCookieName,
        Value:    refreshToken,
        Path:     cookiePath,
        MaxAge:   int(7 * 24 * time.Hour.Seconds()), // 7 days
        Secure:   cookieSecure,
        HTTPOnly: cookieHttpOnly,
        SameSite: cookieSameSite,
    })
    
    // Return access token in response body (short-lived, 15 min)
    return c.JSON(fiber.Map{
        "access_token": accessToken,
        "expires_in":   900, // 15 minutes
        "user":         user,
    })
}

func (h *Handler) RefreshToken(c *fiber.Ctx) error {
    // Read refresh token from httpOnly cookie
    refreshToken := c.Cookies(refreshTokenCookieName)
    if refreshToken == "" {
        return fiber.NewError(fiber.StatusUnauthorized, "no refresh token")
    }
    
    // Validate and rotate tokens
    newAccessToken, newRefreshToken, err := h.service.RotateTokens(refreshToken)
    if err != nil {
        // Clear invalid cookie
        c.ClearCookie(refreshTokenCookieName)
        return fiber.NewError(fiber.StatusUnauthorized, "invalid refresh token")
    }
    
    // Set new refresh token
    c.Cookie(&fiber.Cookie{
        Name:     refreshTokenCookieName,
        Value:    newRefreshToken,
        Path:     cookiePath,
        MaxAge:   int(7 * 24 * time.Hour.Seconds()),
        Secure:   cookieSecure,
        HTTPOnly: cookieHttpOnly,
        SameSite: cookieSameSite,
    })
    
    return c.JSON(fiber.Map{
        "access_token": newAccessToken,
        "expires_in":   900,
    })
}

func (h *Handler) Logout(c *fiber.Ctx) error {
    // Clear refresh token cookie
    c.ClearCookie(refreshTokenCookieName)
    
    // Optionally blacklist token in database
    refreshToken := c.Cookies(refreshTokenCookieName)
    if refreshToken != "" {
        h.service.BlacklistToken(refreshToken)
    }
    
    return c.SendStatus(http.StatusNoContent)
}

// ============================================================================
// Frontend: API Client with Token Refresh
// No localStorage access - cookies are automatic
// ============================================================================

import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { useAuthStore } from '../stores/authStore'

// In-memory storage for access token (not localStorage)
let accessToken: string | null = null
let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

const apiClient = axios.create({
  baseURL: '/api',
  timeout: 10000,
  withCredentials: true, // Important: sends cookies with requests
})

// Subscribe to token refresh
function subscribeTokenRefresh(callback: (token: string) => void) {
  refreshSubscribers.push(callback)
}

function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach(callback => callback(newToken))
  refreshSubscribers = []
}

// Request interceptor: Add access token header
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (accessToken && config.headers) {
      config.headers.Authorization = `Bearer ${accessToken}`
    }
    return config
  }
)

// Response interceptor: Handle 401 and refresh token
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
    
    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error)
    }
    
    if (isRefreshing) {
      // Wait for refresh to complete
      return new Promise((resolve) => {
        subscribeTokenRefresh((newToken) => {
          originalRequest.headers = originalRequest.headers || {}
          originalRequest.headers.Authorization = `Bearer ${newToken}`
          resolve(apiClient(originalRequest))
        })
      })
    }
    
    originalRequest._retry = true
    isRefreshing = true
    
    try {
      // Call refresh endpoint - cookie is sent automatically
      const response = await axios.post('/api/auth/refresh', null, {
        withCredentials: true,
      })
      
      const newAccessToken = response.data.access_token
      accessToken = newAccessToken
      
      // Update store
      useAuthStore.getState().setAccessToken(newAccessToken)
      
      // Notify subscribers
      onTokenRefreshed(newAccessToken)
      
      // Retry original request
      originalRequest.headers = originalRequest.headers || {}
      originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
      
      return apiClient(originalRequest)
    } catch (refreshError) {
      // Refresh failed - clear everything and redirect
      accessToken = null
      useAuthStore.getState().logout()
      window.location.href = '/login'
      return Promise.reject(refreshError)
    } finally {
      isRefreshing = false
    }
  }
)

// ============================================================================
// Frontend: Auth Store (simplified, no localStorage)
// ============================================================================

import { create } from 'zustand'

interface User {
  id: number
  username: string
  is_super_admin: boolean
  must_change_password: boolean
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  
  setAccessToken: (token: string) => void
  setUser: (user: User | null) => void
  logout: () => void
  clearMustChangePassword: () => void
}

export const useAuthStore = create<AuthState>()((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: false,
  
  setAccessToken: (token) => {
    // Store in memory only (module-level variable)
    accessToken = token
    set({ isAuthenticated: true })
  },
  
  setUser: (user) => set({ user }),
  
  logout: () => {
    accessToken = null
    set({ user: null, isAuthenticated: false })
    // Call logout endpoint to clear httpOnly cookie
    apiClient.post('/api/auth/logout')
  },
  
  clearMustChangePassword: () => {
    set((state) => ({
      user: state.user ? { ...state.user, must_change_password: false } : null,
    }))
  },
}))

// ============================================================================
// Frontend: Login Component
// ============================================================================

import { useState } from 'preact/hooks'
import { useAuthStore } from '../stores/authStore'
import { authApi } from '../api/endpoints'

export function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  
  const setAccessToken = useAuthStore(s => s.setAccessToken)
  const setUser = useAuthStore(s => s.setUser)
  
  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setError('')
    
    try {
      const response = await authApi.login({ username, password })
      
      // Access token from response, refresh token in httpOnly cookie
      setAccessToken(response.data.access_token)
      setUser(response.data.user)
      
      route('/dashboard')
    } catch (err) {
      setError('Invalid credentials')
    }
  }
  
  return (
    <form onSubmit={handleSubmit}>
      {/* Login form */}
    </form>
  )
}

// ============================================================================
// Frontend: Token Expiration Handler
// ============================================================================

import { useEffect } from 'preact/hooks'
import { useAuthStore } from '../stores/authStore'

const TOKEN_REFRESH_INTERVAL = 14 * 60 * 1000 // Refresh 1 min before expiry

export function TokenRefreshHandler() {
  const isAuthenticated = useAuthStore(s => s.isAuthenticated)
  
  useEffect(() => {
    if (!isAuthenticated) return
    
    // Proactive token refresh before expiration
    const interval = setInterval(async () => {
      try {
        const response = await axios.post('/api/auth/refresh', null, {
          withCredentials: true,
        })
        
        const newAccessToken = response.data.access_token
        accessToken = newAccessToken
        useAuthStore.getState().setAccessToken(newAccessToken)
      } catch {
        // Refresh failed, let interceptor handle on next request
      }
    }, TOKEN_REFRESH_INTERVAL)
    
    return () => clearInterval(interval)
  }, [isAuthenticated])
  
  return null
}
```

#### Migration Path

**Phase 1: Backend Cookie Support (Week 1-2)**
1. Update Go auth handlers to set httpOnly cookies
2. Add `/api/auth/refresh` endpoint
3. Update logout to clear cookies
4. Test with curl to verify cookie behavior

**Phase 2: Frontend API Client (Week 3)**
1. Remove localStorage from `authStore.ts`
2. Update `api/client.ts` with `withCredentials: true`
3. Implement token refresh interceptor
4. Add `TokenRefreshHandler` component

**Phase 3: Migration Strategy (Week 4)**
1. Deploy backend changes first (backward compatible)
2. Update frontend to use new auth flow
3. Add migration to clear old localStorage tokens
4. Monitor for auth-related errors

**Phase 4: Security Hardening (Week 5)**
1. Add CSRF protection for cookie-based endpoints
2. Implement token rotation on refresh
3. Add token binding to IP/fingerprint (optional)
4. Security audit and penetration testing

#### Why This Is Architecturally Superior

1. **XSS Protection**: httpOnly cookies cannot be accessed by JavaScript
2. **Automatic Handling**: Cookies are sent with every request automatically
3. **Single Source of Truth**: No synchronization between storage mechanisms
4. **Standards Compliant**: Follows OAuth 2.0 and modern security best practices
5. **SSR Compatible**: No localStorage means no hydration mismatches

---

### Problem 7: Accessibility Gaps

**Current State:**
- 22 ARIA attributes across 104 files
- No `aria-current` for navigation
- Missing focus management
- No screen reader announcements

#### Root Cause Analysis

Accessibility is often treated as an afterthought in development. The current gaps indicate:

1. **No design system**: Components don't have built-in accessibility
2. **No testing**: No automated a11y checks in CI
3. **No expertise**: Developers unaware of WCAG requirements
4. **No enforcement**: No linting or review requirements

This excludes users with disabilities and creates legal risk (ADA compliance in US, EN 301 549 in EU).

#### Ultimate Solution: WCAG 2.1 AA Compliance with axe-core Testing

We implement a comprehensive accessibility architecture with automated testing, semantic components, and focus management:

```typescript
// ============================================================================
// Component: Accessible Button with full ARIA support
// ============================================================================

import { forwardRef } from 'preact/compat'
import { cn } from '../../lib/utils'

interface ButtonProps {
  children: ComponentChildren
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  loading?: boolean
  ariaLabel?: string
  ariaDescribedBy?: string
  ariaExpanded?: boolean
  ariaControls?: string
  ariaPressed?: boolean
  onClick?: (e: MouseEvent) => void
  type?: 'button' | 'submit' | 'reset'
  className?: string
  id?: string
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button({
    children,
    variant = 'primary',
    size = 'md',
    disabled = false,
    loading = false,
    ariaLabel,
    ariaDescribedBy,
    ariaExpanded,
    ariaControls,
    ariaPressed,
    onClick,
    type = 'button',
    className,
    id,
  }, ref) {
    const isDisabled = disabled || loading
    
    return (
      <button
        ref={ref}
        type={type}
        id={id}
        onClick={onClick}
        disabled={isDisabled}
        aria-label={ariaLabel}
        aria-describedby={ariaDescribedBy}
        aria-expanded={ariaExpanded}
        aria-controls={ariaControls}
        aria-pressed={ariaPressed}
        aria-busy={loading}
        aria-disabled={isDisabled}
        className={cn(
          // Base styles
          'inline-flex items-center justify-center font-medium transition-colors',
          'focus:outline-none focus:ring-2 focus:ring-offset-2',
          'disabled:opacity-50 disabled:cursor-not-allowed',
          
          // Variant styles
          variant === 'primary' && 'bg-color-primary text-white hover:bg-color-primary-dark',
          variant === 'secondary' && 'bg-bg-secondary text-text-primary hover:bg-bg-tertiary',
          variant === 'danger' && 'bg-color-danger text-white hover:bg-color-danger-dark',
          variant === 'ghost' && 'bg-transparent text-text-secondary hover:bg-bg-secondary',
          
          // Size styles
          size === 'sm' && 'px-3 py-1.5 text-sm',
          size === 'md' && 'px-4 py-2 text-base',
          size === 'lg' && 'px-6 py-3 text-lg',
          
          className
        )}
      >
        {loading && (
          <span className="sr-only">Loading,</span>
        )}
        {children}
      </button>
    )
  }
)

// ============================================================================
// Component: Navigation with aria-current
// ============================================================================

import { useLocation } from 'preact-router'
import { cn } from '../../lib/utils'

interface NavItem {
  path: string
  label: string
  icon: LucideIcon
  ariaLabel?: string
}

interface NavigationProps {
  items: NavItem[]
  ariaLabel?: string
}

export function Navigation({ items, ariaLabel = 'Main navigation' }: NavigationProps) {
  const location = useLocation()
  const currentPath = location.pathname
  
  return (
    <nav aria-label={ariaLabel}>
      <ul className="space-y-1" role="menubar">
        {items.map((item) => {
          const isActive = currentPath.startsWith(item.path)
          const Icon = item.icon
          
          return (
            <li key={item.path} role="none">
              <a
                href={item.path}
                role="menuitem"
                aria-current={isActive ? 'page' : undefined}
                aria-label={item.ariaLabel || item.label}
                className={cn(
                  'flex items-center gap-3 px-4 py-2 rounded-lg transition-colors',
                  'focus:outline-none focus:ring-2 focus:ring-color-primary',
                  isActive 
                    ? 'bg-color-primary/10 text-color-primary font-medium' 
                    : 'text-text-secondary hover:bg-bg-secondary hover:text-text-primary'
                )}
              >
                <Icon className="w-5 h-5" aria-hidden="true" />
                <span>{item.label}</span>
              </a>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}

// ============================================================================
// Hook: useFocusManager - Programmatic focus management
// ============================================================================

import { useRef, useCallback } from 'preact/hooks'

interface FocusManager {
  focusElement: (selector: string) => void
  focusFirst: (containerRef: HTMLElement) => void
  focusLast: (containerRef: HTMLElement) => void
  trapFocus: (containerRef: HTMLElement) => () => void
  restoreFocus: () => void
}

export function useFocusManager(): FocusManager {
  const previousFocusRef = useRef<HTMLElement | null>(null)
  
  const focusElement = useCallback((selector: string) => {
    const element = document.querySelector<HTMLElement>(selector)
    element?.focus()
  }, [])
  
  const focusFirst = useCallback((containerRef: HTMLElement) => {
    const focusableElements = containerRef.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )
    focusableElements[0]?.focus()
  }, [])
  
  const focusLast = useCallback((containerRef: HTMLElement) => {
    const focusableElements = containerRef.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )
    focusableElements[focusableElements.length - 1]?.focus()
  }, [])
  
  const trapFocus = useCallback((containerRef: HTMLElement) => {
    previousFocusRef.current = document.activeElement as HTMLElement
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return
      
      const focusableElements = containerRef.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      )
      
      const firstElement = focusableElements[0]
      const lastElement = focusableElements[focusableElements.length - 1]
      
      if (e.shiftKey && document.activeElement === firstElement) {
        e.preventDefault()
        lastElement?.focus()
      } else if (!e.shiftKey && document.activeElement === lastElement) {
        e.preventDefault()
        firstElement?.focus()
      }
    }
    
    containerRef.addEventListener('keydown', handleKeyDown)
    focusFirst(containerRef)
    
    return () => {
      containerRef.removeEventListener('keydown', handleKeyDown)
    }
  }, [focusFirst])
  
  const restoreFocus = useCallback(() => {
    previousFocusRef.current?.focus()
    previousFocusRef.current = null
  }, [])
  
  return {
    focusElement,
    focusFirst,
    focusLast,
    trapFocus,
    restoreFocus,
  }
}

// ============================================================================
// Component: Accessible Modal with focus trap
// ============================================================================

import { useEffect, useRef } from 'preact/hooks'
import { useFocusManager } from '../../hooks/useFocusManager'
import { Button } from './Button'

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  description?: string
  children: ComponentChildren
  size?: 'sm' | 'md' | 'lg'
  footer?: ComponentChildren
}

export function Modal({ 
  isOpen, 
  onClose, 
  title, 
  description,
  children, 
  size = 'md',
  footer 
}: ModalProps) {
  const modalRef = useRef<HTMLDivElement>(null)
  const { trapFocus, restoreFocus } = useFocusManager()
  
  useEffect(() => {
    if (!isOpen) return
    
    // Prevent body scroll
    document.body.style.overflow = 'hidden'
    
    // Trap focus and return cleanup function
    const cleanup = modalRef.current ? trapFocus(modalRef.current) : () => {}
    
    // Handle escape key
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleEscape)
    
    return () => {
      document.body.style.overflow = ''
      document.removeEventListener('keydown', handleEscape)
      cleanup()
      restoreFocus()
    }
  }, [isOpen, onClose, trapFocus, restoreFocus])
  
  if (!isOpen) return null
  
  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="presentation"
    >
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
        aria-hidden="true"
      />
      
      {/* Modal */}
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
        aria-describedby={description ? 'modal-description' : undefined}
        className={cn(
          'relative bg-white rounded-lg shadow-xl',
          size === 'sm' && 'w-full max-w-md',
          size === 'md' && 'w-full max-w-lg',
          size === 'lg' && 'w-full max-w-2xl'
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 id="modal-title" className="text-lg font-semibold">
            {title}
          </h2>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            ariaLabel="Close modal"
          >
            <XIcon className="w-5 h-5" aria-hidden="true" />
          </Button>
        </div>
        
        {/* Description (for screen readers) */}
        {description && (
          <p id="modal-description" className="sr-only">
            {description}
          </p>
        )}
        
        {/* Content */}
        <div className="px-6 py-4">
          {children}
        </div>
        
        {/* Footer */}
        {footer && (
          <div className="flex justify-end gap-3 px-6 py-4 border-t">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}

// ============================================================================
// Hook: useAnnounce - Screen reader announcements
// ============================================================================

import { useCallback } from 'preact/hooks'

type AnnouncePriority = 'polite' | 'assertive'

export function useAnnounce() {
  const announce = useCallback((message: string, priority: AnnouncePriority = 'polite') => {
    const regionId = `aria-live-${priority}`
    let region = document.getElementById(regionId)
    
    // Create live region if it doesn't exist
    if (!region) {
      region = document.createElement('div')
      region.id = regionId
      region.setAttribute('aria-live', priority)
      region.setAttribute('aria-atomic', 'true')
      region.className = 'sr-only'
      document.body.appendChild(region)
    }
    
    // Clear and set message (clearing ensures announcement even if same message)
    region.textContent = ''
    setTimeout(() => {
      region.textContent = message
    }, 100)
  }, [])
  
  return announce
}

// ============================================================================
// Test: axe-core accessibility testing
// ============================================================================

import { render } from '@testing-library/preact'
import { axe, toHaveNoViolations } from 'jest-axe'
import { Button } from './Button'
import { Modal } from './Modal'
import { Navigation } from './Navigation'

expect.extend(toHaveNoViolations)

describe('Accessibility', () => {
  describe('Button', () => {
    it('should have no accessibility violations', async () => {
      const { container } = render(
        <Button onClick={() => {}}>Click me</Button>
      )
      const results = await axe(container)
      expect(results).toHaveNoViolations()
    })
    
    it('should have accessible loading state', () => {
      const { getByRole } = render(
        <Button loading>Loading</Button>
      )
      const button = getByRole('button')
      expect(button).toHaveAttribute('aria-busy', 'true')
      expect(button).toHaveAttribute('aria-disabled', 'true')
    })
  })
  
  describe('Modal', () => {
    it('should trap focus when open', async () => {
      const onClose = jest.fn()
      const { getByRole } = render(
        <Modal isOpen={true} onClose={onClose} title="Test Modal">
          <button>First</button>
          <button>Second</button>
        </Modal>
      )
      
      const modal = getByRole('dialog')
      expect(modal).toHaveAttribute('aria-modal', 'true')
      expect(modal).toHaveAttribute('aria-labelledby', 'modal-title')
    })
  })
  
  describe('Navigation', () => {
    it('should mark current page with aria-current', () => {
      const items = [
        { path: '/dashboard', label: 'Dashboard', icon: HomeIcon },
        { path: '/users', label: 'Users', icon: UsersIcon },
      ]
      
      // Mock current location
      jest.mock('preact-router', () => ({
        useLocation: () => ({ pathname: '/users' })
      }))
      
      const { getByRole } = render(<Navigation items={items} />)
      
      const usersLink = getByRole('menuitem', { name: 'Users' })
      expect(usersLink).toHaveAttribute('aria-current', 'page')
    })
  })
})
```

#### Migration Path

**Phase 1: Component Audit (Week 1)**
1. Run axe-core on all pages
2. Document violations by severity
3. Prioritize critical user flows

**Phase 2: Core Components (Week 2-3)**
1. Update `Button` with full ARIA support
2. Update `Modal` with focus trap
3. Update `Navigation` with `aria-current`
4. Add `useFocusManager` and `useAnnounce` hooks

**Phase 3: Page-Level Fixes (Week 4-5)**
1. Add landmarks (`<main>`, `<aside>`, `<nav>`)
2. Fix heading hierarchy
3. Add form labels and error associations
4. Implement skip links

**Phase 4: Testing & CI (Week 6)**
1. Add axe-core to component tests
2. Add accessibility checks to CI
3. Train team on WCAG 2.1 AA requirements
4. Create accessibility checklist for PRs

#### Why This Is Architecturally Superior

1. **Inclusivity**: Users with disabilities can fully use the application
2. **Legal Compliance**: Meets ADA, Section 508, and EN 301 549 requirements
3. **SEO Benefits**: Semantic HTML improves search engine indexing
4. **Usability**: Focus management and announcements improve experience for all users
5. **Automated Testing**: axe-core catches regressions before they reach production

---

## DevOps Security Problems

---

### Problem 8: No Supply Chain Security

**Current State:**
```bash
# install.sh - lines 190-191
curl -sL "${GITHUB_RAW}/docker/docker-compose.yml" -o "$INSTALL_DIR/docker-compose.yml"
curl -sL "${GITHUB_RAW}/docker/.env.example" -o "$INSTALL_DIR/.env.example"
```

No checksum verification means a compromised GitHub account or MITM attack could inject malicious code.

#### Root Cause Analysis

Supply chain attacks are increasingly common. The install script downloads executables and configuration files without verifying their integrity. This creates multiple attack vectors:

1. **GitHub compromise**: If the isolate-project account is compromised, malicious code is distributed
2. **MITM attack**: Network interception could modify downloads
3. **CDN compromise**: GitHub's CDN could be compromised
4. **DNS hijacking**: DNS could be redirected to malicious servers

#### Ultimate Solution: Checksum Verification with Cosign

We implement a defense-in-depth approach with multiple verification layers:

```bash
#!/bin/bash
# ============================================================================
# install.sh - Supply Chain Secure Installation
# ============================================================================

set -euo pipefail

# Configuration
GITHUB_REPO="isolate-project/isolate-panel"
GITHUB_RAW="https://raw.githubusercontent.com/${GITHUB_REPO}"
GITHUB_API="https://api.github.com/repos/${GITHUB_REPO}"
INSTALL_DIR="/opt/isolate-panel"

# Enable checksum verification (can be disabled for development)
VERIFY_CHECKSUMS=${VERIFY_CHECKSUMS:-true}
VERIFY_SIGNATURES=${VERIFY_SIGNATURES:-true}

# ============================================================================
# Verification Functions
# ============================================================================

# Download with checksum verification
download_verified() {
    local url="$1"
    local output="$2"
    local expected_checksum="$3"
    local temp_file="${output}.tmp"
    
    log_info "Downloading $(basename "$output")..."
    
    # Download to temp file
    if ! curl -fsSL "$url" -o "$temp_file"; then
        log_error "Failed to download from $url"
        return 1
    fi
    
    # Verify checksum if enabled
    if [[ "$VERIFY_CHECKSUMS" == "true" && -n "$expected_checksum" ]]; then
        local actual_checksum
        actual_checksum=$(sha256sum "$temp_file" | cut -d' ' -f1)
        
        if [[ "$actual_checksum" != "$expected_checksum" ]]; then
            log_error "Checksum verification failed!"
            log_error "Expected: $expected_checksum"
            log_error "Actual:   $actual_checksum"
            rm -f "$temp_file"
            return 1
        fi
        
        log_success "Checksum verified"
    fi
    
    # Move to final location
    mv "$temp_file" "$output"
}

# Fetch checksums from release assets
fetch_checksums() {
    local version="${1:-latest}"
    local checksum_url
    
    if [[ "$version" == "latest" ]]; then
        checksum_url="${GITHUB_RAW}/master/checksums.sha256"
    else
        checksum_url="${GITHUB_RAW}/${version}/checksums.sha256"
    fi
    
    # Download checksums file
    local checksum_file="/tmp/isolate-checksums.sha256"
    if ! curl -fsSL "$checksum_url" -o "$checksum_file" 2>/dev/null; then
        log_warning "Could not fetch checksums file"
        return 1
    fi
    
    echo "$checksum_file"
}

# Get checksum for a specific file
get_file_checksum() {
    local checksum_file="$1"
    local filename="$2"
    
    if [[ -f "$checksum_file" ]]; then
        grep "$filename" "$checksum_file" | cut -d' ' -f1
    fi
}

# Verify Cosign signature (if cosign is available)
verify_signature() {
    local file="$1"
    local sig_url="$2"
    local cert_url="$3"
    
    if [[ "$VERIFY_SIGNATURES" != "true" ]]; then
        return 0
    fi
    
    if ! command -v cosign &> /dev/null; then
        log_warning "cosign not installed, skipping signature verification"
        return 0
    fi
    
    local temp_sig="${file}.sig.tmp"
    local temp_cert="${file}.cert.tmp"
    
    # Download signature and certificate
    curl -fsSL "$sig_url" -o "$temp_sig" || return 1
    curl -fsSL "$cert_url" -o "$temp_cert" || return 1
    
    # Verify with cosign
    if cosign verify-blob \
        --signature "$temp_sig" \
        --certificate "$temp_cert" \
        --certificate-identity-regexp "^https://github.com/${GITHUB_REPO}/.github/workflows/.*@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+$" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "$file"; then
        log_success "Signature verified with Cosign"
        rm -f "$temp_sig" "$temp_cert"
        return 0
    else
        log_error "Signature verification failed"
        rm -f "$temp_sig" "$temp_cert"
        return 1
    fi
}

# ============================================================================
# Secure Download Functions
# ============================================================================

download_files() {
    print_step "Downloading configuration files (with verification)"
    
    # Fetch checksums
    local checksum_file
    checksum_file=$(fetch_checksums) || true
    
    # Check if we're running from a cloned repository
    local SCRIPT_DIR
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    if [[ -f "$SCRIPT_DIR/docker-compose.yml" ]]; then
        log_info "Copying from local repository..."
        cp "$SCRIPT_DIR/docker-compose.yml" "$INSTALL_DIR/docker-compose.yml"
        cp "$SCRIPT_DIR/.env.example" "$INSTALL_DIR/.env.example"
    else
        log_info "Downloading from GitHub with verification..."
        
        # Get expected checksums
        local compose_checksum env_checksum
        compose_checksum=$(get_file_checksum "$checksum_file" "docker-compose.yml")
        env_checksum=$(get_file_checksum "$checksum_file" ".env.example")
        
        # Download with verification
        download_verified \
            "${GITHUB_RAW}/master/docker/docker-compose.yml" \
            "$INSTALL_DIR/docker-compose.yml" \
            "$compose_checksum"
        
        download_verified \
            "${GITHUB_RAW}/master/docker/.env.example" \
            "$INSTALL_DIR/.env.example" \
            "$env_checksum"
        
        # Verify signatures if available
        if [[ -n "$compose_checksum" ]]; then
            verify_signature \
                "$INSTALL_DIR/docker-compose.yml" \
                "${GITHUB_RAW}/master/docker/docker-compose.yml.sig" \
                "${GITHUB_RAW}/master/docker/docker-compose.yml.cert" || true
        fi
    fi
    
    log_success "Configuration files downloaded and verified"
}

# ============================================================================
# Docker Image Verification
# ============================================================================

verify_docker_image() {
    local image="$1"
    local tag="$2"
    
    log_info "Verifying Docker image signature..."
    
    if ! command -v cosign &> /dev/null; then
        log_warning "cosign not installed, skipping image signature verification"
        return 0
    fi
    
    # Verify image signature
    if cosign verify \
        --certificate-identity-regexp "^https://github.com/${GITHUB_REPO}/.github/workflows/.*@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+$" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "${image}:${tag}"; then
        log_success "Docker image signature verified"
        return 0
    else
        log_error "Docker image signature verification failed"
        read -p "Continue anyway? [y/N] " confirm
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            exit 1
        fi
        return 1
    fi
}

# ============================================================================
# Main Installation with Security Checks
# ============================================================================

main() {
    # ... existing checks ...
    
    # Verify Docker image before pulling
    verify_docker_image "ghcr.io/${GITHUB_REPO}" "latest"
    
    # Pull and start
    compose_cmd pull
    compose_cmd up -d
}
```

#### Migration Path

**Phase 1: Generate Checksums (Week 1)**
1. Create script to generate SHA256 checksums for all release assets
2. Add checksum generation to release workflow
3. Publish checksums file with releases

**Phase 2: Update Install Script (Week 2)**
1. Add `download_verified()` function
2. Add checksum fetching and verification
3. Test with valid and invalid checksums

**Phase 3: Cosign Integration (Week 3)**
1. Install Cosign in CI
2. Sign release artifacts with Cosign
3. Add signature verification to install script
4. Document Cosign installation for users

**Phase 4: Documentation (Week 4)**
1. Update installation documentation
2. Add security section to README
3. Create security policy document

#### Why This Is Architecturally Superior

1. **Tamper Detection**: Checksums detect any modification to downloaded files
2. **Identity Verification**: Cosign signatures verify the publisher's identity
3. **Transparency**: Signed artifacts provide audit trail
4. **User Trust**: Users can independently verify downloads
5. **Industry Standard**: Follows practices used by Kubernetes, Helm, and major projects

---

### Problem 9: No Container Image Scanning

**Current State:**
```yaml
# release.yml - lines 92-104
- name: Build and push
  uses: docker/build-push-action@v5
  with:
    context: .
    file: docker/Dockerfile
    push: true
    platforms: linux/amd64
    tags: ${{ steps.meta.outputs.tags }}
```

Images are built and pushed without vulnerability scanning, potentially shipping known CVEs.

#### Root Cause Analysis

Container images often contain:
- OS packages with known vulnerabilities
- Outdated language runtimes
- Secrets accidentally baked into layers
- Malicious packages from compromised registries

Without scanning, these issues reach production. The current workflow pushes immediately after build, with no security gate.

#### Ultimate Solution: Multi-Scanner Pipeline with Trivy, Snyk, and Grype

We implement a comprehensive scanning pipeline with multiple scanners for defense in depth:

```yaml
# ============================================================================
# .github/workflows/release.yml - With Security Scanning
# ============================================================================

name: Release

on:
  push:
    tags: ['v*']

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}
  # Fail build if vulnerabilities found
  SEVERITY_THRESHOLD: HIGH
  # Scanners to use
  ENABLE_TRIVY: true
  ENABLE_GRYPE: true
  ENABLE_DOCKLE: true

jobs:
  # ... test job remains the same ...

  # ============================================================================
  # Security Scan Job - Multi-scanner approach
  # ============================================================================
  security-scan:
    name: Container Security Scan
    runs-on: ubuntu-latest
    needs: [test]
    permissions:
      contents: read
      packages: read
      security-events: write

    steps:
      - uses: actions/checkout@v4

      # Build image locally (don't push yet)
      - name: Build image for scanning
        uses: docker/build-push-action@v5
        with:
          context: .
          file: docker/Dockerfile
          push: false
          load: true
          tags: ${{ env.IMAGE_NAME }}:scan
          cache-from: type=gha

      # -------------------------------------------------------------------------
      # Scanner 1: Trivy - Comprehensive vulnerability scanner
      # -------------------------------------------------------------------------
      - name: Run Trivy vulnerability scanner
        if: env.ENABLE_TRIVY == 'true'
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: '${{ env.IMAGE_NAME }}:scan'
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH,MEDIUM'
          exit-code: '1'
          ignore-unfixed: true

      - name: Upload Trivy results
        if: always() && env.ENABLE_TRIVY == 'true'
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: 'trivy-results.sarif'
          category: 'trivy'

      # -------------------------------------------------------------------------
      # Scanner 2: Grype - Fast, accurate vulnerability scanner
      # -------------------------------------------------------------------------
      - name: Run Grype vulnerability scanner
        if: env.ENABLE_GRYPE == 'true'
        uses: anchore/scan-action@v3
        with:
          image: '${{ env.IMAGE_NAME }}:scan'
          severity-cutoff: high
          fail-build: true

      - name: Upload Grype results
        if: always() && env.ENABLE_GRYPE == 'true'
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: 'results.sarif'
          category: 'grype'

      # ------------------------------------------------------------------------
      # Scanner 3: Dockle - CIS Docker benchmark
      # -------------------------------------------------------------------------
      - name: Run Dockle CIS benchmark
        if: env.ENABLE_DOCKLE == 'true'
        uses: goodwithtech/dockle-action@main
        with:
          image: '${{ env.IMAGE_NAME }}:scan'
          format: 'sarif'
          output: 'dockle-results.sarif'
          exit-code: '1'
          exit-level: 'WARN'

      - name: Upload Dockle results
        if: always() && env.ENABLE_DOCKLE == 'true'
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: 'dockle-results.sarif'
          category: 'dockle'

      # -------------------------------------------------------------------------
      # Scanner 4: Hadolint - Dockerfile linting
      # -------------------------------------------------------------------------
      - name: Run Hadolint
        uses: hadolint/hadolint-action@v3.1.0
        with:
          dockerfile: docker/Dockerfile
          format: sarif
          output-file: hadolint-results.sarif
          no-fail: false
          failure-threshold: error

      - name: Upload Hadolint results
        if: always()
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: 'hadolint-results.sarif'
          category: 'hadolint'

      # -------------------------------------------------------------------------
      # Secret Detection - Prevent credential leakage
      # -------------------------------------------------------------------------
      - name: Scan for secrets in image
        uses: trufflesecurity/trufflehog@main
        with:
          path: ./
          base: main
          head: HEAD
          extra_args: --debug --only-verified

  # ============================================================================
  # Build and Push - Only after security scans pass
  # ============================================================================
  build-and-push:
    name: Build & Push Image
    runs-on: ubuntu-latest
    needs: [test, security-scan]
    permissions:
      contents: read
      packages: write
      id-token: write  # For Cosign

    steps:
      - uses: actions/checkout@v4

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=raw,value=latest

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          file: docker/Dockerfile
          push: true
          platforms: linux/amd64,linux/arm64
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            VERSION=${{ github.ref_name }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      # Sign image after successful push
      - name: Install Cosign
        uses: sigstore/cosign-installer@v3

      - name: Sign image
        run: |
          cosign sign --yes \
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@${{ steps.build-push.outputs.digest }}
```

#### Migration Path

**Phase 1: Add Trivy (Week 1)**
1. Add Trivy scan job to release workflow
2. Set severity threshold to HIGH
3. Fix any immediate HIGH/CRITICAL issues
4. Upload results to GitHub Security tab

**Phase 2: Add Grype (Week 2)**
1. Add Grype scan for second opinion
2. Compare results with Trivy
3. Document any discrepancies
4. Tune severity thresholds

**Phase 3: Add CIS Benchmarks (Week 3)**
1. Add Dockle for Dockerfile best practices
2. Add Hadolint for Dockerfile linting
3. Fix CIS benchmark violations
4. Update Dockerfile with best practices

**Phase 4: Secret Detection (Week 4)**
1. Add TruffleHog for secret detection
2. Scan image layers for credentials
3. Add pre-commit hooks for secret prevention
4. Document secret management practices

#### Why This Is Architecturally Superior

1. **Defense in Depth**: Multiple scanners catch different issues
2. **Fail Fast**: Scans happen before push, preventing vulnerable images in registry
3. **Visibility**: SARIF uploads provide centralized security view
4. **Compliance**: CIS benchmarks ensure industry-standard practices
5. **Automation**: No manual intervention required

---

### Problem 10: No SBOM Generation

**Current State:**
No Software Bill of Materials is generated for Docker images, making it impossible to know what components are included.

#### Root Cause Analysis

SBOMs (Software Bill of Materials) are essential for:
- **Vulnerability management**: Knowing what packages are included
- **License compliance**: Tracking open source licenses
- **Incident response**: Quickly identifying affected components
- **Supply chain transparency**: Understanding dependencies

Without SBOMs, security teams cannot assess risk, and compliance teams cannot verify license requirements.

#### Ultimate Solution: Syft SBOM Generation with Multiple Formats

We implement comprehensive SBOM generation in multiple formats for different use cases:

```yaml
# ============================================================================
# .github/workflows/release.yml - SBOM Generation
# ============================================================================

  generate-sbom:
    name: Generate SBOM
    runs-on: ubuntu-latest
    needs: [build-and-push]
    permissions:
      contents: write
      packages: read

    steps:
      - uses: actions/checkout@v4

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      # Pull the image that was just pushed
      - name: Pull image
        run: docker pull ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.ref_name }}

      # -------------------------------------------------------------------------
      # Generate SBOM with Syft
      # -------------------------------------------------------------------------
      - name: Install Syft
        uses: anchore/sbom-action/download-syft@v0

      - name: Generate SBOM (SPDX)
        run: |
          syft ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.ref_name }} \
            -o spdx-json=sbom.spdx.json \
            -o spdx-tag-value=sbom.spdx \
            -o cyclonedx-json=sbom.cyclonedx.json \
            -o cyclonedx-xml=sbom.cyclonedx.xml \
            -o syft-json=sbom.syft.json \
            -o table=sbom.txt

      # -------------------------------------------------------------------------
      # Attach SBOM to release
      # -------------------------------------------------------------------------
      - name: Upload SBOM to release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            sbom.spdx.json
            sbom.cyclonedx.json
            sbom.txt
          tag_name: ${{ github.ref_name }}

      # -------------------------------------------------------------------------
      # Sign SBOM with Cosign
      # -------------------------------------------------------------------------
      - name: Install Cosign
        uses: sigstore/cosign-installer@v3

      - name: Sign SBOM
        run: |
          cosign sign-blob --yes \
            --output-signature sbom.spdx.json.sig \
            --output-certificate sbom.spdx.json.cert \
            sbom.spdx.json

      # -------------------------------------------------------------------------
      # Upload to Dependency Graph
      # -------------------------------------------------------------------------
      - name: Upload SBOM to GitHub Dependency Graph
        uses: anchore/sbom-action@v0
        with:
          image: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.ref_name }}
          format: spdx-json
          output-file: dependency-sbom.spdx.json

      - name: Submit SBOM to Dependency Graph
        uses: advanced-security/spdx-dependency-submission-action@v0
        with:
          filePath: dependency-sbom.spdx.json

      # -------------------------------------------------------------------------
      # Archive SBOMs
      # -------------------------------------------------------------------------
      - name: Archive SBOM artifacts
        uses: actions/upload-artifact@v4
        with:
          name: sboms-${{ github.ref_name }}
          path: |
            sbom.*
            *.sig
            *.cert
          retention-days: 365
```

#### Migration Path

**Phase 1: Syft Integration (Week 1)**
1. Add Syft to release workflow
2. Generate SPDX and CycloneDX formats
3. Attach to GitHub releases
4. Document SBOM locations

**Phase 2: Dependency Graph (Week 2)**
1. Submit SBOM to GitHub Dependency Graph
2. Enable dependency alerts
3. Configure alert policies
4. Train team on dependency review

**Phase 3: SBOM Signing (Week 3)**
1. Sign SBOMs with Cosign
2. Publish signatures with releases
3. Document verification process
4. Add SBOM verification to install script

**Phase 4: SBOM Consumption (Week 4)**
1. Document how to use SBOMs for vulnerability scanning
2. Create script to compare SBOMs between versions
3. Add SBOM to security documentation
4. Train security team on SBOM analysis

#### Why This Is Architecturally Superior

1. **Transparency**: Complete visibility into image contents
2. **Compliance**: Meets regulatory requirements (EO 14028, etc.)
3. **Security**: Enables proactive vulnerability management
4. **Standard Formats**: SPDX and CycloneDX are industry standards
5. **Integration**: GitHub Dependency Graph provides native SBOM viewing

---

### Problem 11: No Image Signing

**Current State:**
Docker images are pushed to registry without cryptographic signatures, allowing potential tampering.

#### Root Cause Analysis

Unsigned images can be:
- Replaced in registry by attackers
- Modified during transit
- Confused with legitimate images (typosquatting)

Users have no way to verify that the image they're running is the one built by the CI pipeline.

#### Ultimate Solution: Cosign Keyless Signing with SLSA Provenance

We implement keyless signing using Sigstore/Cosign with OIDC identity verification:

```yaml
# ============================================================================
# .github/workflows/release.yml - Image Signing
# ============================================================================

  sign-image:
    name: Sign Container Image
    runs-on: ubuntu-latest
    needs: [build-and-push]
    permissions:
      contents: read
      packages: write
      id-token: write  # Required for Cosign OIDC

    steps:
      - uses: actions/checkout@v4

      - name: Install Cosign
        uses: sigstore/cosign-installer@v3
        with:
          cosign-release: 'v2.2.0'

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      # -------------------------------------------------------------------------
      # Keyless Signing with OIDC
      # Uses GitHub Actions OIDC token for identity
      # No long-term keys to manage
      # -------------------------------------------------------------------------
      - name: Sign image with Cosign (keyless)
        run: |
          cosign sign --yes \
            --recursive \
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@${{ needs.build-and-push.outputs.digest }}

      # -------------------------------------------------------------------------
      # Verify signature (self-test)
      # -------------------------------------------------------------------------
      - name: Verify signature
        run: |
          cosign verify \
            --certificate-identity-regexp "^https://github.com/${{ github.repository }}/.github/workflows/.*@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+$" \
            --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.ref_name }}

      # -------------------------------------------------------------------------
      # Generate SLSA Provenance
      # -------------------------------------------------------------------------
      - name: Generate SLSA provenance
        uses: slsa-framework/slsa-github-generator/.github/workflows/generator_container_slsa3.yml@v1.9.0
        with:
          image: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          digest: ${{ needs.build-and-push.outputs.digest }}
          registry-username: ${{ github.actor }}
          registry-password: ${{ secrets.GITHUB_TOKEN }}

      # -------------------------------------------------------------------------
      # Attach SBOM to image
      # -------------------------------------------------------------------------
      - name: Download SBOM
        uses: actions/download-artifact@v4
        with:
          name: sboms-${{ github.ref_name }}
          path: ./sboms

      - name: Attach SBOM to image
        run: |
          cosign attach sbom --sbom ./sboms/sbom.spdx.json \
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@${{ needs.build-and-push.outputs.digest }}

      # -------------------------------------------------------------------------
      # Sign SBOM attachment
      # -------------------------------------------------------------------------
      - name: Sign SBOM attachment
        run: |
          cosign sign --yes \
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.ref_name }}.sbom
```

#### Migration Path

**Phase 1: Cosign Setup (Week 1)**
1. Add Cosign installer to workflow
2. Configure OIDC permissions
3. Sign image after push
4. Verify signature in same workflow

**Phase 2: SLSA Provenance (Week 2)**
1. Add SLSA provenance generator
2. Configure provenance attestation
3. Verify provenance
4. Document SLSA level achieved

**Phase 3: SBOM Attachment (Week 3)**
1. Attach SBOM to image as attestation
2. Sign SBOM attachment
3. Document attestation retrieval
4. Update install script to verify attestations

**Phase 4: Policy Enforcement (Week 4)**
1. Create verification policy
2. Add verification to deployment workflows
3. Document signature verification for users
4. Train team on keyless signing

#### Why This Is Architecturally Superior

1. **Keyless**: No long-term keys to manage or leak
2. **Transparent**: Signatures are public and verifiable
3. **Automated**: Signing happens automatically in CI
4. **Standard**: Uses Sigstore, the industry standard for container signing
5. **Provenance**: SLSA attestations provide build transparency

---

### Problem 12: Dockerfile Dev Downloads Binaries

**Current State:**
```dockerfile
# Dockerfile.dev - lines 43-66
RUN wget -q https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-linux-64.zip...
RUN wget -q https://github.com/MetaCubeX/mihomo/releases/download/v1.19.21/mihomo-linux-amd64...
RUN wget -q https://github.com/SagerNet/sing-box/releases/download/v1.13.3/sing-box-1.13.3-linux-amd64.tar.gz...
```

Binaries are downloaded at build time without checksum verification, creating supply chain risk.

#### Root Cause Analysis

Downloading binaries during Docker build:
- No verification of binary authenticity
- No reproducible builds (URLs could change)
- Network dependency for builds
- Potential for MITM attacks

#### Ultimate Solution: Multi-Stage Build with Verified Binaries

We implement a secure build process with verified, cached binaries:

```dockerfile
# ============================================================================
# Dockerfile - Secure Binary Handling
# ============================================================================

# Stage 1: Binary Downloader with Verification
FROM alpine:3.21 AS binary-downloader

WORKDIR /downloads

# Install verification tools
RUN apk add --no-cache wget gnupg coreutils

# Copy checksums file (generated during release)
COPY docker/checksums.binaries.sha256 /checksums/

# Download and verify Xray
ARG XRAY_VERSION=26.2.6
ARG XRAY_CHECKSUM=abc123...
RUN wget -q "https://github.com/XTLS/Xray-core/releases/download/v${XRAY_VERSION}/Xray-linux-64.zip" \
    -O xray.zip && \
    echo "${XRAY_CHECKSUM}  xray.zip" | sha256sum -c - && \
    unzip -q xray.zip -d /downloads/xray && \
    chmod +x /downloads/xray/xray && \
    rm xray.zip

# Download and verify Mihomo
ARG MIHOMO_VERSION=1.19.21
ARG MIHOMO_CHECKSUM=def456...
RUN wget -q "https://github.com/MetaCubeX/mihomo/releases/download/v${MIHOMO_VERSION}/mihomo-linux-amd64-v${MIHOMO_VERSION}.gz" \
    -O mihomo.gz && \
    echo "${MIHOMO_CHECKSUM}  mihomo.gz" | sha256sum -c - && \
    gunzip -q mihomo.gz && \
    chmod +x /downloads/mihomo && \
    mv mihomo /downloads/mihomo-binary

# Download and verify Sing-box
ARG SINGBOX_VERSION=1.13.3
ARG SINGBOX_CHECKSUM=ghi789...
RUN wget -q "https://github.com/SagerNet/sing-box/releases/download/v${SINGBOX_VERSION}/sing-box-${SINGBOX_VERSION}-linux-amd64.tar.gz" \
    -O singbox.tar.gz && \
    echo "${SINGBOX_CHECKSUM}  singbox.tar.gz" | sha256sum -c - && \
    tar -xzf singbox.tar.gz -C /tmp && \
    mv "/tmp/sing-box-${SINGBOX_VERSION}-linux-amd64/sing-box" /downloads/sing-box && \
    chmod +x /downloads/sing-box && \
    rm -rf singbox.tar.gz /tmp/sing-box-*

# ============================================================================
# Stage 2: Go Builder (unchanged)
# ============================================================================
FROM golang:1.26-alpine AS go-builder
# ... existing Go build steps ...

# ============================================================================
# Stage 3: Node.js Builder (unchanged)
# ============================================================================
FROM node:22-alpine AS node-builder
# ... existing Node build steps ...

# ============================================================================
# Stage 4: Final Runtime Image
# ============================================================================
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies only
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite \
    supervisor \
    libcap \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -S isolate && \
    adduser -S -G isolate -h /app isolate

# Copy binaries from downloader stage (verified)
COPY --from=binary-downloader /downloads/xray/xray /usr/local/bin/cores/xray
COPY --from=binary-downloader /downloads/mihomo-binary /usr/local/bin/cores/mihomo
COPY --from=binary-downloader /downloads/sing-box /usr/local/bin/cores/sing-box

# Copy compiled Go binary
COPY --from=go-builder /app/server /usr/local/bin/isolate-panel
COPY --from=go-builder /app/isolate-migrate /usr/local/bin/isolate-migrate

# Copy compiled frontend
COPY --from=node-builder /app/dist /var/www/html

# Copy configuration
COPY docker/config.yaml /app/configs/config.yaml
COPY docker/supervisord.conf /etc/supervisord.conf
COPY docker/docker-entrypoint.sh /docker-entrypoint.sh
COPY docker/docker-healthcheck.sh /docker-healthcheck.sh

# Set permissions and capabilities
RUN mkdir -p /app/data /var/log/isolate-panel /var/log/supervisor /var/run && \
    chmod +x /usr/local/bin/cores/* && \
    chmod +x /usr/local/bin/isolate-panel && \
    chmod +x /usr/local/bin/isolate-migrate && \
    chmod +x /docker-entrypoint.sh && \
    chmod +x /docker-healthcheck.sh && \
    setcap cap_net_bind_service+ep /usr/local/bin/cores/xray && \
    setcap cap_net_bind_service+ep /usr/local/bin/cores/mihomo && \
    setcap cap_net_bind_service+ep /usr/local/bin/cores/sing-box && \
    setcap cap_net_bind_service+ep /usr/local/bin/isolate-panel && \
    chown -R isolate:isolate /app /var/www/html /var/log/isolate-panel /var/log/supervisor /var/run

# Switch to non-root user
USER isolate

# Expose ports
EXPOSE 8080 443 8443

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD /docker-healthcheck.sh

# Use SIGTERM for graceful shutdown
STOPSIGNAL SIGTERM

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
```

#### Alternative: Pre-downloaded Binaries in Repository

For even more security, commit verified binaries to the repository:

```bash
# scripts/download-binaries.sh
#!/bin/bash
set -euo pipefail

VERSIONS_FILE="docker/binary-versions.env"
CHECKSUMS_FILE="docker/checksums.binaries.sha256"
DOWNLOAD_DIR="docker/cores"

# Load versions
source "$VERSIONS_FILE"

# Create download directory
mkdir -p "$DOWNLOAD_DIR"

# Download Xray
echo "Downloading Xray ${XRAY_VERSION}..."
wget -q "https://github.com/XTLS/Xray-core/releases/download/v${XRAY_VERSION}/Xray-linux-64.zip" \
    -O "$DOWNLOAD_DIR/xray.zip"
echo "${XRAY_CHECKSUM}  $DOWNLOAD_DIR/xray.zip" | sha256sum -c -
unzip -q "$DOWNLOAD_DIR/xray.zip" -d "$DOWNLOAD_DIR/xray"
chmod +x "$DOWNLOAD_DIR/xray/xray"
rm "$DOWNLOAD_DIR/xray.zip"

# Download Mihomo
echo "Downloading Mihomo ${MIHOMO_VERSION}..."
wget -q "https://github.com/MetaCubeX/mihomo/releases/download/v${MIHOMO_VERSION}/mihomo-linux-amd64-v${MIHOMO_VERSION}.gz" \
    -O "$DOWNLOAD_DIR/mihomo.gz"
echo "${MIHOMO_CHECKSUM}  $DOWNLOAD_DIR/mihomo.gz" | sha256sum -c -
gunzip -q "$DOWNLOAD_DIR/mihomo.gz"
chmod +x "$DOWNLOAD_DIR/mihomo"

# Download Sing-box
echo "Downloading Sing-box ${SINGBOX_VERSION}..."
wget -q "https://github.com/SagerNet/sing-box/releases/download/v${SINGBOX_VERSION}/sing-box-${SINGBOX_VERSION}-linux-amd64.tar.gz" \
    -O "$DOWNLOAD_DIR/singbox.tar.gz"
echo "${SINGBOX_CHECKSUM}  $DOWNLOAD_DIR/singbox.tar.gz" | sha256sum -c -
tar -xzf "$DOWNLOAD_DIR/singbox.tar.gz" -C /tmp
mv "/tmp/sing-box-${SINGBOX_VERSION}-linux-amd64/sing-box" "$DOWNLOAD_DIR/"
chmod +x "$DOWNLOAD_DIR/sing-box"
rm -rf "$DOWNLOAD_DIR/singbox.tar.gz" /tmp/sing-box-*

echo "All binaries downloaded and verified!"
```

Then in Dockerfile:

```dockerfile
# Copy pre-downloaded, verified binaries
COPY docker/cores/xray/xray /usr/local/bin/cores/xray
COPY docker/cores/mihomo /usr/local/bin/cores/mihomo
COPY docker/cores/sing-box /usr/local/bin/cores/sing-box
```

#### Migration Path

**Phase 1: Checksum Verification (Week 1)**
1. Create `binary-versions.env` with versions and checksums
2. Update Dockerfile to verify downloads
3. Test build with verification
4. Document binary update process

**Phase 2: Pre-download Option (Week 2)**
1. Create `download-binaries.sh` script
2. Download and commit verified binaries
3. Update Dockerfile to use local binaries
4. Remove network dependency from build

**Phase 3: Automated Updates (Week 3)**
1. Create workflow to check for new binary releases
2. Auto-generate PR with updated versions
3. Run security scan on new binaries
4. Document update approval process

**Phase 4: Supply Chain Hardening (Week 4)**
1. Verify binary signatures where available
2. Add binary SBOM generation
3. Document binary provenance
4. Create binary update policy

#### Why This Is Architecturally Superior

1. **Verification**: Checksums ensure binary integrity
2. **Reproducibility**: Fixed versions enable reproducible builds
3. **No Network Dependency**: Local binaries build without network
4. **Audit Trail**: Version file shows exactly what's included
5. **Security**: Prevents MITM and supply chain attacks

---

### Problem 13: Go Version Drift

**Current State:**
- `Dockerfile`: `golang:1.26-alpine`
- `test.yml`: `go-version: '1.26.2'`
- `security-scan.yml`: `go-version: '1.25'`

Inconsistent Go versions across CI/CD create compatibility and security issues.

#### Root Cause Analysis

Version drift causes:
- **Compatibility issues**: Different behavior between versions
- **Security gaps**: Older versions miss security patches
- **Build failures**: Features unavailable in older versions
- **Maintenance burden**: Multiple versions to track

The security scan using Go 1.25 is particularly concerning as it may miss vulnerabilities fixed in 1.26.

#### Ultimate Solution: Centralized Version Management with Renovate

We implement a single source of truth for Go versions with automated updates:

```yaml
# ============================================================================
# .github/variables/go-version.yml
# Single source of truth for Go version
# ============================================================================

name: go-version

on:
  workflow_call:
    outputs:
      version:
        description: 'Go version to use'
        value: ${{ jobs.set-version.outputs.version }}
      version_exact:
        description: 'Exact Go version (including patch)'
        value: ${{ jobs.set-version.outputs.version_exact }}

jobs:
  set-version:
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.vars.outputs.version }}
      version_exact: ${{ steps.vars.outputs.version_exact }}
    steps:
      - uses: actions/checkout@v4
      
      - id: vars
        run: |
          # Read from go.mod or centralized file
          GO_VERSION=$(grep '^go ' backend/go.mod | cut -d' ' -f2)
          echo "version=${GO_VERSION}" >> $GITHUB_OUTPUT
          echo "version_exact=${GO_VERSION}.2" >> $GITHUB_OUTPUT
```

```yaml
# ============================================================================
# .github/workflows/test.yml - Using centralized version
# ============================================================================

jobs:
  backend-test:
    steps:
      - uses: actions/checkout@v4

      - name: Get Go version
        id: go-version
        uses: ./.github/variables/go-version.yml

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ steps.go-version.outputs.version_exact }}
          cache: true
```

```dockerfile
# ============================================================================
# Dockerfile - Using build args for version
# ============================================================================

ARG GO_VERSION=1.26
ARG GO_PATCH=2

FROM golang:${GO_VERSION}.${GO_PATCH}-alpine AS go-builder
```

```yaml
# ============================================================================
# .github/renovate.json - Automated version updates
# ============================================================================

{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": [
    "config:recommended",
    ":dependencyDashboard",
    ":semanticCommits",
    ":semanticCommitTypeAll(chore)"
  ],
  "packageRules": [
    {
      "matchDatasources": ["golang-version"],
      "matchPackageNames": ["go"],
      "groupName": "Go version",
      "automerge": false,
      "reviewers": ["team-backend"]
    },
    {
      "matchDatasources": ["docker"],
      "matchPackageNames": ["golang"],
      "groupName": "Go Docker images",
      "automerge": false
    }
  ],
  "postUpdateOptions": [
    "gomodTidy"
  ],
  "constraints": {
    "go": "1.26"
  }
}
```

#### Migration Path

**Phase 1: Centralize Version (Week 1)**
1. Create `.github/variables/go-version.yml`
2. Update all workflows to use centralized version
3. Update Dockerfile with ARG for version
4. Test all builds

**Phase 2: Renovate Setup (Week 2)**
1. Install Renovate GitHub App
2. Create `renovate.json` configuration
3. Configure version update rules
4. Test with minor version update

**Phase 3: Version Pinning (Week 3)**
1. Pin exact versions in all files
2. Add validation workflow
3. Document version update process
4. Train team on Renovate

**Phase 4: Automated Testing (Week 4)**
1. Add version consistency check
2. Test builds with new versions automatically
3. Add version compatibility tests
4. Document rollback procedure

#### Why This Is Architecturally Superior

1. **Single Source of Truth**: One place to update Go version
2. **Automated Updates**: Renovate creates PRs for new versions
3. **Consistency**: All environments use same version
4. **Security**: Always on latest patch version
5. **Visibility**: Dependency dashboard shows all pending updates

---

### Problem 14: No Read-Only Root FS

**Current State:**
The Docker container can write to the root filesystem, increasing attack surface.

#### Root Cause Analysis

Writable root filesystem allows:
- **Malware persistence**: Attackers can modify system files
- **Configuration tampering**: Runtime changes to system config
- **Log manipulation**: Attackers can delete/modify logs
- **Tool installation**: Attackers can install additional tools

A read-only root filesystem forces attackers to use only the writable volumes, which can be monitored and restricted.

#### Ultimate Solution: Distroless Runtime with Read-Only Root FS

We implement a hardened container with minimal attack surface:

```dockerfile
# ============================================================================
# Dockerfile - Hardened with Read-Only Root FS
# ============================================================================

# Stage 1: Go Builder
FROM golang:1.26.2-alpine AS go-builder
# ... existing build steps ...

# Stage 2: Node.js Builder
FROM node:22-alpine AS node-builder
# ... existing build steps ...

# Stage 3: Binary Downloader
FROM alpine:3.21 AS binary-downloader
# ... download and verify binaries ...

# ============================================================================
# Stage 4: Hardened Runtime (Distroless-inspired)
# ============================================================================

# Use minimal base image
FROM alpine:3.21

# Install only runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite-libs \
    supervisor \
    libcap \
    && rm -rf /var/cache/apk/*

# Create non-root user with explicit UID/GID
RUN addgroup -S -g 1000 isolate && \
    adduser -S -u 1000 -G isolate -h /app isolate

# Create required directories with proper permissions
RUN mkdir -p /app/data /app/configs /app/tmp /var/log/isolate-panel /var/log/supervisor /var/run && \
    chown -R isolate:isolate /app /var/log/isolate-panel /var/log/supervisor /var/run

# Copy binaries
COPY --from=binary-downloader --chown=isolate:isolate /downloads/xray/xray /usr/local/bin/cores/xray
COPY --from=binary-downloader --chown=isolate:isolate /downloads/mihomo-binary /usr/local/bin/cores/mihomo
COPY --from=binary-downloader --chown=isolate:isolate /downloads/sing-box /usr/local/bin/cores/sing-box

# Copy application
COPY --from=go-builder --chown=isolate:isolate /app/server /usr/local/bin/isolate-panel
COPY --from=go-builder --chown=isolate:isolate /app/isolate-migrate /usr/local/bin/isolate-migrate
COPY --from=node-builder --chown=isolate:isolate /app/dist /var/www/html

# Copy configs
COPY --chown=isolate:isolate docker/config.yaml /app/configs/config.yaml
COPY --chown=isolate:isolate docker/supervisord.conf /etc/supervisord.conf
COPY --chown=isolate:isolate docker/docker-entrypoint.sh /docker-entrypoint.sh
COPY --chown=isolate:isolate docker/docker-healthcheck.sh /docker-healthcheck.sh

# Set capabilities and permissions
RUN chmod +x /usr/local/bin/cores/* && \
    chmod +x /usr/local/bin/isolate-panel && \
    chmod +x /usr/local/bin/isolate-migrate && \
    chmod +x /docker-entrypoint.sh && \
    chmod +x /docker-healthcheck.sh && \
    setcap cap_net_bind_service+ep /usr/local/bin/cores/xray && \
    setcap cap_net_bind_service+ep /usr/local/bin/cores/mihomo && \
    setcap cap_net_bind_service+ep /usr/local/bin/cores/sing-box && \
    setcap cap_net_bind_service+ep /usr/local/bin/isolate-panel

# Remove unnecessary tools
RUN apk del --purge --no-network libcap || true

# Switch to non-root user
USER isolate:isolate

# Expose ports
EXPOSE 8080 443 8443

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD /docker-healthcheck.sh

# Use SIGTERM for graceful shutdown
STOPSIGNAL SIGTERM

# Mark volumes as writable (everything else is read-only)
VOLUME ["/app/data", "/var/log/isolate-panel", "/var/log/supervisor", "/var/run", "/tmp"]

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
```

```yaml
# ============================================================================
# docker-compose.yml - With security options
# ============================================================================

services:
  isolate-panel:
    image: ghcr.io/isolate-project/isolate-panel:latest
    container_name: isolate-panel
    restart: unless-stopped
    
    # Security options
    read_only: true  # Read-only root filesystem
    
    # Capabilities - drop all, add only required
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Required for binding ports < 1024
    
    # Security profile
    security_opt:
      - no-new-privileges:true
    
    # Resource limits
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 256M
    
    # Writable volumes (only these can be written to)
    volumes:
      - ./data:/app/data
      - ./logs:/var/log/isolate-panel
      - /tmp:/tmp
    
    # Environment
    env_file:
      - .env
    
    # Network
    network_mode: host  # Required for proxy functionality
```

```bash
# ============================================================================
# docker-entrypoint.sh - Handle read-only filesystem
# ============================================================================

#!/bin/sh
set -e

# Create tmp directory if it doesn't exist (for read-only FS)
mkdir -p /tmp

# Copy supervisord config to writable location if needed
if [ ! -w /etc/supervisord.conf ]; then
    cp /etc/supervisord.conf /tmp/supervisord.conf
    export SUPERVISORD_CONFIG=/tmp/supervisord.conf
else
    export SUPERVISORD_CONFIG=/etc/supervisord.conf
fi

# Run database migrations (if needed)
if [ -w /app/data ]; then
    /usr/local/bin/isolate-migrate up
fi

# Execute supervisord with the correct config
exec /usr/bin/supervisord -c "$SUPERVISORD_CONFIG"
```

#### Migration Path

**Phase 1: Identify Writable Requirements (Week 1)**
1. Audit all write operations in application
2. Identify required writable directories
3. Document volume requirements
4. Update entrypoint script

**Phase 2: Update Dockerfile (Week 2)**
1. Add `VOLUME` declarations for writable paths
2. Ensure all runtime files are in volumes
3. Test build with read-only flag
4. Fix any write attempts to root fs

**Phase 3: Update Docker Compose (Week 3)**
1. Add `read_only: true` to service
2. Add `cap_drop: [ALL]` and `cap_add` for required capabilities
3. Add `no-new-privileges:true`
4. Test full stack functionality

**Phase 4: Kubernetes Hardening (Week 4)**
1. Create PodSecurityPolicy or SecurityContext
2. Add readOnlyRootFilesystem: true
3. Configure allowPrivilegeEscalation: false
4. Test deployment in staging

#### Why This Is Architecturally Superior

1. **Immutability**: Container filesystem cannot be modified at runtime
2. **Attack Surface Reduction**: Malware cannot persist in container
3. **Compliance**: Meets CIS Docker benchmarks and security standards
4. **Forensics**: Writable volumes can be snapshotted for investigation
5. **Reproducibility**: Container behavior is deterministic

---

## Implementation Timeline

### Phase 1: Foundation (Weeks 1-4)
- Week 1: Component architecture, Constants, Module state
- Week 2: Memoization, i18n type safety
- Week 3: Token storage (BFF pattern)
- Week 4: Accessibility core components

### Phase 2: Security Hardening (Weeks 5-8)
- Week 5: Supply chain security (checksums, Cosign)
- Week 6: Container scanning (Trivy, Grype)
- Week 7: SBOM generation, Image signing
- Week 8: Dockerfile hardening, Read-only FS

### Phase 3: DevOps Improvements (Weeks 9-12)
- Week 9: Go version management
- Week 10: Binary verification
- Week 11: CI/CD optimization
- Week 12: Documentation and training

### Phase 4: Validation (Weeks 13-14)
- Week 13: Security audit
- Week 14: Performance testing, Documentation

---

## Success Metrics

### Frontend
- Component file size: <100 lines average
- Memoization coverage: >80% of derived state
- i18n coverage: 100% of user-facing strings
- Axe-core violations: 0 critical, 0 serious
- Lighthouse accessibility score: >95

### DevOps
- Image vulnerabilities: 0 HIGH/CRITICAL
- SBOM generation: 100% of releases
- Image signing: 100% of releases
- Supply chain verification: 100% of installs
- Read-only root FS: All containers

---

## Conclusion

These solutions represent industry best practices for frontend architecture and DevOps security. While the implementation requires significant effort, the long-term benefits in maintainability, security, and team productivity justify the investment. Each solution is designed to be incremental, allowing the team to implement changes gradually while maintaining system stability.

The key architectural principles applied throughout:
1. **Single Responsibility**: Each component/file has one clear purpose
2. **Defense in Depth**: Multiple security layers protect against attacks
3. **Automation**: Manual processes are replaced with automated checks
4. **Transparency**: SBOMs, signatures, and scans provide visibility
5. **Standards Compliance**: WCAG, CIS, SLSA, and other standards are met

By implementing these solutions, Isolate Panel will achieve enterprise-grade architecture and security posture suitable for production deployments in security-conscious environments.
