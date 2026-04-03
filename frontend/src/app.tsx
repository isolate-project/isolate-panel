import { Suspense, lazy } from 'preact/compat'
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

// Wrapper components for protected routes
function ProtectedDashboard() {
  return (
    <ProtectedRoute>
      <Dashboard />
    </ProtectedRoute>
  )
}

function ProtectedUsers() {
  return (
    <ProtectedRoute>
      <Users />
    </ProtectedRoute>
  )
}

function ProtectedCores() {
  return (
    <ProtectedRoute>
      <Cores />
    </ProtectedRoute>
  )
}

function ProtectedInbounds() {
  return (
    <ProtectedRoute>
      <Inbounds />
    </ProtectedRoute>
  )
}

function ProtectedInboundCreate() {
  return (
    <ProtectedRoute>
      <InboundCreate />
    </ProtectedRoute>
  )
}

function ProtectedInboundDetail({ id }: { id?: string }) {
  return (
    <ProtectedRoute>
      <InboundDetail id={Number(id)} />
    </ProtectedRoute>
  )
}

function ProtectedInboundEdit({ id }: { id?: string }) {
  return (
    <ProtectedRoute>
      <InboundEdit id={Number(id)} />
    </ProtectedRoute>
  )
}

function ProtectedOutbounds() {
  return (
    <ProtectedRoute>
      <Outbounds />
    </ProtectedRoute>
  )
}

function ProtectedCertificates() {
  return (
    <ProtectedRoute>
      <Certificates />
    </ProtectedRoute>
  )
}

function ProtectedActiveConnections() {
  return (
    <ProtectedRoute>
      <ActiveConnections />
    </ProtectedRoute>
  )
}

function ProtectedSettings() {
  return (
    <ProtectedRoute>
      <Settings />
    </ProtectedRoute>
  )
}

function ProtectedWarpRoutes() {
  return (
    <ProtectedRoute>
      <WarpRoutes />
    </ProtectedRoute>
  )
}

function ProtectedGeoRules() {
  return (
    <ProtectedRoute>
      <GeoRules />
    </ProtectedRoute>
  )
}

function ProtectedBackups() {
  return (
    <ProtectedRoute>
      <Backups />
    </ProtectedRoute>
  )
}

function ProtectedNotifications() {
  return (
    <ProtectedRoute>
      <Notifications />
    </ProtectedRoute>
  )
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
            <Route path="/" component={ProtectedDashboard} />
            <Route path="/users" component={ProtectedUsers} />
            <Route path="/cores" component={ProtectedCores} />
            <Route path="/inbounds" component={ProtectedInbounds} />
            <Route path="/inbounds/create" component={ProtectedInboundCreate} />
            <Route path="/inbounds/:id/edit" component={ProtectedInboundEdit} />
            <Route path="/inbounds/:id" component={ProtectedInboundDetail} />
            <Route path="/outbounds" component={ProtectedOutbounds} />
            <Route path="/certificates" component={ProtectedCertificates} />
            <Route path="/connections" component={ProtectedActiveConnections} />
            <Route path="/settings" component={ProtectedSettings} />
            <Route path="/warp" component={ProtectedWarpRoutes} />
            <Route path="/geo" component={ProtectedGeoRules} />
            <Route path="/backups" component={ProtectedBackups} />
            <Route path="/notifications" component={ProtectedNotifications} />
            <Route default component={NotFound} />
          </Router>
        </Suspense>
      </ErrorBoundary>
      <ToastContainer />
    </>
  )
}
