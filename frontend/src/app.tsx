import { Suspense, lazy, ComponentType } from 'preact/compat'
import { Router, Route } from 'preact-router'
import { ErrorBoundary } from './components/layout/ErrorBoundary'
import { ProtectedRoute } from './router/ProtectedRoute'
import { ToastContainer } from './components/ui/ToastContainer'
import { PageSkeleton } from './components/ui/PageSkeleton'
import { useSessionExpired } from './hooks/useSessionExpired'
import './i18n'

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
  return (
    <ProtectedRoute>
      <Component {...rest} />
    </ProtectedRoute>
  )
}

// Wrappers for routes that need prop transforms (path params → typed props)
function ProtectedInboundDetail({ id }: { id?: string }) {
  return <Protected Component={InboundDetail} id={Number(id)} />
}

function ProtectedInboundEdit({ id }: { id?: string }) {
  return <Protected Component={InboundEdit} id={Number(id)} />
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
