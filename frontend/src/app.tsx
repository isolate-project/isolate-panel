import { Router, Route } from 'preact-router'
import { ProtectedRoute } from './router/ProtectedRoute'
import { ToastContainer } from './components/ui/ToastContainer'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { Users } from './pages/Users'
import { Cores } from './pages/Cores'
import { Inbounds } from './pages/Inbounds'
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
        <Route path="/settings" component={ProtectedSettings} />
        <Route default component={NotFound} />
      </Router>
      <ToastContainer />
    </>
  )
}
