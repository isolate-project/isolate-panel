import { Router, Route } from 'preact-router'
import { ProtectedRoute } from './router/ProtectedRoute'
import { ToastContainer } from './components/ui/ToastContainer'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { Users } from './pages/Users'
import { Cores } from './pages/Cores'
import { Inbounds } from './pages/Inbounds'
import { InboundCreate } from './pages/InboundCreate'
import { InboundDetail } from './pages/InboundDetail'
import { InboundEdit } from './pages/InboundEdit'
import { Outbounds } from './pages/Outbounds'
import { Certificates } from './pages/Certificates'
import { ActiveConnections } from './pages/ActiveConnections'
import { Settings } from './pages/Settings'
import { NotFound } from './pages/NotFound'
import { useSessionExpired } from './hooks/useSessionExpired'
import './i18n'

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

export function App() {
  // Monitor for session expiration
  useSessionExpired()

  return (
    <>
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
        <Route default component={NotFound} />
      </Router>
      <ToastContainer />
    </>
  )
}
