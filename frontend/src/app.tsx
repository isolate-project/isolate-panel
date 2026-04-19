import { Suspense, lazy, ComponentType } from 'preact/compat'
import { Router, Route } from 'preact-router'
import { ErrorBoundary } from './components/layout/ErrorBoundary'
import { ProtectedRoute } from './router/ProtectedRoute'
import { ToastContainer } from './components/ui/ToastContainer'
import { PageSkeleton } from './components/ui/PageSkeleton'
import { useSessionExpired } from './hooks/useSessionExpired'
import './i18n'

<<<<<<< Updated upstream
const Login = lazy(() => import('./pages/Login').then(m => ({ default: m.Login })))
const Dashboard = lazy(() => import('./pages/Dashboard').then(m => ({ default: m.Dashboard })))
const Users = lazy(() => import('./pages/Users').then(m => ({ default: m.Users })))
const Cores = lazy(() => import('./pages/Cores').then(m => ({ default: m.Cores })))
const Inbounds = lazy(() => import('./pages/Inbounds').then(m => ({ default: m.Inbounds })))
const InboundCreate = lazy(() => import('./pages/InboundCreate').then(m => ({ default: m.InboundCreate })))
const InboundDetail = lazy(() => import('./pages/InboundDetail').then(m => ({ default: m.InboundDetail })))
const InboundEdit = lazy(() => import('./pages/InboundEdit').then(m => ({ default: m.InboundEdit })))
const Outbounds = lazy(() => import('./pages/Outbounds').then(m => ({ default: m.Outbounds })))
const Certificates = lazy(() => import('./pages/Certificates').then(m => ({ default: m.Certificates })))
const ActiveConnections = lazy(() => import('./pages/ActiveConnections').then(m => ({ default: m.ActiveConnections })))
const Settings = lazy(() => import('./pages/Settings').then(m => ({ default: m.Settings })))
const WarpRoutes = lazy(() => import('./pages/WarpRoutes').then(m => ({ default: m.WarpRoutes })))
const GeoRules = lazy(() => import('./pages/GeoRules').then(m => ({ default: m.GeoRules })))
const Backups = lazy(() => import('./pages/Backups').then(m => ({ default: m.Backups })))
const Notifications = lazy(() => import('./pages/Notifications').then(m => ({ default: m.Notifications })))
const NotFound = lazy(() => import('./pages/NotFound').then(m => ({ default: m.NotFound })))

// Generic protected route wrapper — eliminates 15 identical wrapper components
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function Protected({ Component, ...rest }: { Component: ComponentType<any>;[key: string]: any }) {
=======
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
const ChangePassword = lazy(retryLazy(() => import('./pages/ChangePassword'), 'ChangePassword'))

function Protected({ Component, ...rest }: { Component: ComponentType<Record<string, unknown>>;[key: string]: unknown }) {
>>>>>>> Stashed changes
  return (
    <ProtectedRoute>
      <Component {...rest} />
    </ProtectedRoute>
  )
}

// Wrappers for routes that need prop transforms (path params → typed props)
function ProtectedInboundDetail({ id }: { id?: string }) {
<<<<<<< Updated upstream
  return <Protected Component={InboundDetail} id={Number(id)} />
}

function ProtectedInboundEdit({ id }: { id?: string }) {
  return <Protected Component={InboundEdit} id={Number(id)} />
}

=======
  const numId = id ? Number(id) : 0
  if (!numId || isNaN(numId)) return <NotFound />
  return <Protected Component={InboundDetail as unknown as ComponentType<Record<string, unknown>>} id={numId} />
}

function ProtectedInboundEdit({ id }: { id?: string }) {
  const numId = id ? Number(id) : 0
  if (!numId || isNaN(numId)) return <NotFound />
  return <Protected Component={InboundEdit as unknown as ComponentType<Record<string, unknown>>} id={numId} />
}

const EBInboundDetail = withErrorBoundary(ProtectedInboundDetail as unknown as ComponentType<Record<string, unknown>>)
const EBInboundEdit = withErrorBoundary(ProtectedInboundEdit as unknown as ComponentType<Record<string, unknown>>)

function withErrorBoundary(Component: ComponentType<Record<string, unknown>>) {
  return function ErrorBoundaryWrapper(props: Record<string, unknown>) {
    return (
      <ErrorBoundary>
        <Component {...props} />
      </ErrorBoundary>
    )
  }
}

function ProtectedDashboard() { return <Protected Component={Dashboard} /> }
function ProtectedUsers() { return <Protected Component={Users} /> }
function ProtectedCores() { return <Protected Component={Cores} /> }
function ProtectedInbounds() { return <Protected Component={Inbounds} /> }
function ProtectedInboundCreate() { return <Protected Component={InboundCreate} /> }
function ProtectedOutbounds() { return <Protected Component={Outbounds} /> }
function ProtectedCertificates() { return <Protected Component={Certificates} /> }
function ProtectedActiveConnections() { return <Protected Component={ActiveConnections} /> }
function ProtectedSettings() { return <Protected Component={Settings} /> }
function ProtectedWarpRoutes() { return <Protected Component={WarpRoutes} /> }
function ProtectedGeoRules() { return <Protected Component={GeoRules} /> }
function ProtectedBackups() { return <Protected Component={Backups} /> }
function ProtectedNotifications() { return <Protected Component={Notifications} /> }

function ProtectedChangePassword() { return <Protected Component={ChangePassword} /> }

const EBChangePassword = withErrorBoundary(ProtectedChangePassword)
const EBDashboard = withErrorBoundary(ProtectedDashboard)
const EBUsers = withErrorBoundary(ProtectedUsers)
const EBCores = withErrorBoundary(ProtectedCores)
const EBInbounds = withErrorBoundary(ProtectedInbounds)
const EBInboundCreate = withErrorBoundary(ProtectedInboundCreate)
const EBOutbounds = withErrorBoundary(ProtectedOutbounds)
const EBCertificates = withErrorBoundary(ProtectedCertificates)
const EBActiveConnections = withErrorBoundary(ProtectedActiveConnections)
const EBSettings = withErrorBoundary(ProtectedSettings)
const EBWarpRoutes = withErrorBoundary(ProtectedWarpRoutes)
const EBGeoRules = withErrorBoundary(ProtectedGeoRules)
const EBBackups = withErrorBoundary(ProtectedBackups)
const EBNotifications = withErrorBoundary(ProtectedNotifications)

const EBNotFound = withErrorBoundary(NotFound)

>>>>>>> Stashed changes
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
