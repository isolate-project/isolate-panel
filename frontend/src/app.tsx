import { Suspense, lazy, ComponentType } from 'preact/compat'
import { Router, Route } from 'preact-router'
import { ErrorBoundary } from './components/layout/ErrorBoundary'
import { ProtectedRoute } from './router/ProtectedRoute'
import { ToastContainer } from './components/ui/ToastContainer'
import { PageSkeleton } from './components/ui/PageSkeleton'
import { useSessionExpired } from './hooks/useSessionExpired'
import './i18n'

function retryLazy<T extends ComponentType<Record<string, unknown>>>(
  importFn: () => Promise<{ [key: string]: T }>,
  exportName: string,
  retries = 3,
  delay = 1000,
): () => Promise<{ default: T }> {
  return () => {
    const attempt = (n: number): Promise<{ default: T }> =>
      importFn().then(m => ({ default: m[exportName] })).catch(err => {
        if (n >= retries) throw err
        return new Promise<{ default: T }>(resolve =>
          setTimeout(() => resolve(attempt(n + 1)), delay * n),
        )
      })
    return attempt(1)
  }
}

const Login = lazy(retryLazy(() => import('./pages/Login'), 'Login'))
const Dashboard = lazy(retryLazy(() => import('./pages/Dashboard'), 'Dashboard'))
const Users = lazy(retryLazy(() => import('./pages/Users'), 'Users'))
const Cores = lazy(retryLazy(() => import('./pages/Cores'), 'Cores'))
const Inbounds = lazy(retryLazy(() => import('./pages/Inbounds'), 'Inbounds'))
const InboundCreate = lazy(retryLazy(() => import('./pages/InboundCreate'), 'InboundCreate'))
const InboundDetail = lazy(retryLazy(() => import('./pages/InboundDetail') as unknown as Promise<{ [key: string]: ComponentType<Record<string, unknown>> }>, 'InboundDetail'))
const InboundEdit = lazy(retryLazy(() => import('./pages/InboundEdit') as unknown as Promise<{ [key: string]: ComponentType<Record<string, unknown>> }>, 'InboundEdit'))
const Outbounds = lazy(retryLazy(() => import('./pages/Outbounds'), 'Outbounds'))
const Certificates = lazy(retryLazy(() => import('./pages/Certificates'), 'Certificates'))
const ActiveConnections = lazy(retryLazy(() => import('./pages/ActiveConnections'), 'ActiveConnections'))
const Settings = lazy(retryLazy(() => import('./pages/Settings'), 'Settings'))
const WarpRoutes = lazy(retryLazy(() => import('./pages/WarpRoutes'), 'WarpRoutes'))
const GeoRules = lazy(retryLazy(() => import('./pages/GeoRules'), 'GeoRules'))
const Backups = lazy(retryLazy(() => import('./pages/Backups'), 'Backups'))
const Notifications = lazy(retryLazy(() => import('./pages/Notifications'), 'Notifications'))
const NotFound = lazy(retryLazy(() => import('./pages/NotFound'), 'NotFound'))

function Protected({ Component, ...rest }: { Component: ComponentType<Record<string, unknown>>;[key: string]: unknown }) {
  return (
    <ProtectedRoute>
      <Component {...rest} />
    </ProtectedRoute>
  )
}

// Wrappers for routes that need prop transforms (path params → typed props)
function ProtectedInboundDetail({ id }: { id?: string }) {
  const numId = id ? Number(id) : 0
  if (!numId || isNaN(numId)) return <NotFound />
  return <Protected Component={InboundDetail as unknown as ComponentType<Record<string, unknown>>} id={numId} />
}

function ProtectedInboundEdit({ id }: { id?: string }) {
  const numId = id ? Number(id) : 0
  if (!numId || isNaN(numId)) return <NotFound />
  return <Protected Component={InboundEdit as unknown as ComponentType<Record<string, unknown>>} id={numId} />
}
export function App() {
  // Monitor for session expiration
  useSessionExpired()

  return (
    <>
      <ErrorBoundary>
        <Suspense fallback={<PageSkeleton />}>
          <Router>
            <Route path="/login" component={Login} />
            <Route path="/" component={() => <Protected Component={Dashboard} />} />
            <Route path="/users" component={() => <Protected Component={Users} />} />
            <Route path="/cores" component={() => <Protected Component={Cores} />} />
            <Route path="/inbounds" component={() => <Protected Component={Inbounds} />} />
            <Route path="/inbounds/create" component={() => <Protected Component={InboundCreate} />} />
            <Route path="/inbounds/:id/edit" component={ProtectedInboundEdit} />
            <Route path="/inbounds/:id" component={ProtectedInboundDetail} />
            <Route path="/outbounds" component={() => <Protected Component={Outbounds} />} />
            <Route path="/certificates" component={() => <Protected Component={Certificates} />} />
            <Route path="/connections" component={() => <Protected Component={ActiveConnections} />} />
            <Route path="/settings" component={() => <Protected Component={Settings} />} />
            <Route path="/warp" component={() => <Protected Component={WarpRoutes} />} />
            <Route path="/geo" component={() => <Protected Component={GeoRules} />} />
            <Route path="/backups" component={() => <Protected Component={Backups} />} />
            <Route path="/notifications" component={() => <Protected Component={Notifications} />} />
            <Route default component={NotFound} />
          </Router>
        </Suspense>
      </ErrorBoundary>
      <ToastContainer />
    </>
  )
}
