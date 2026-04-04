import { useState, useEffect } from 'preact/hooks'
import { notificationApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'

interface Notification {
  id: number
  event_type: string
  severity: string
  title: string
  message: string
  status: string
  created_at: string
}

export function Notifications() {
  const { addToast } = useToastStore()
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)
  const [showSettings, setShowSettings] = useState(false)
  const [testChannel, setTestChannel] = useState<'webhook' | 'telegram' | 'all'>('all')
  const [sendingTest, setSendingTest] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const notificationsRes = await notificationApi.list()
      setNotifications(notificationsRes.data.data || [])
    } catch (error) {
      console.error('Failed to load notifications:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteNotification = async (id: number) => {
    if (!confirm('Delete this notification?')) return

    try {
      await notificationApi.delete(id)
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to delete: ' + (error.response?.data?.error || error.message) })
    }
  }

  const handleSendTest = async () => {
    setSendingTest(true)
    try {
      await notificationApi.sendTest(testChannel)
      addToast({ type: 'success', message: 'Test notification sent!' })
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to send test: ' + (error.response?.data?.error || error.message) })
    } finally {
      setSendingTest(false)
    }
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
      case 'error': return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300'
      case 'warning': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300'
      case 'info': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
      default: return 'bg-secondary text-secondary'
    }
  }

  const getEventTypeIcon = (eventType: string) => {
    const icons: Record<string, string> = {
      'quota_exceeded': '📊',
      'expiry_warning': '⏰',
      'cert_renewed': '🔒',
      'core_error': '⚙️',
      'failed_login': '🔐',
      'user_created': '👤',
      'user_deleted': '🗑️',
      'test': '🧪',
    }
    return icons[eventType] || '📢'
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-primary">Notifications</h1>
          <p className="text-secondary mt-1">System notifications and alerts</p>
        </div>
        <button
          onClick={() => setShowSettings(!showSettings)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          {showSettings ? 'Hide Settings' : 'Settings'}
        </button>
      </div>

      {/* Settings Panel */}
      {showSettings && (
        <div className="bg-surface border border-primary rounded-lg p-6 mb-6">
          <h2 className="text-lg font-semibold text-primary mb-4">Notification Settings</h2>

          {/* Webhook Settings */}
          <div className="mb-6">
            <h3 className="text-sm font-medium text-secondary mb-2">Webhook</h3>
            <div className="grid grid-cols-2 gap-4">
              <input
                type="text"
                placeholder="Webhook URL"
                className="px-3 py-2 bg-surface border border-primary rounded-lg text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              />
              <input
                type="password"
                placeholder="Webhook Secret"
                className="px-3 py-2 bg-surface border border-primary rounded-lg text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
          </div>

          {/* Telegram Settings */}
          <div className="mb-6">
            <h3 className="text-sm font-medium text-secondary mb-2">Telegram</h3>
            <div className="grid grid-cols-2 gap-4">
              <input
                type="text"
                placeholder="Bot Token"
                className="px-3 py-2 bg-surface border border-primary rounded-lg text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              />
              <input
                type="text"
                placeholder="Chat ID"
                className="px-3 py-2 bg-surface border border-primary rounded-lg text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
          </div>

          {/* Test Notification */}
          <div className="mb-6">
            <h3 className="text-sm font-medium text-secondary mb-2">Test Notification</h3>
            <div className="flex gap-4">
              <select
                value={testChannel}
                onChange={(e) => setTestChannel((e.target as HTMLSelectElement).value as 'telegram' | 'webhook')}
                className="px-3 py-2 bg-surface border border-primary rounded-lg text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option value="all">All Channels</option>
                <option value="webhook">Webhook Only</option>
                <option value="telegram">Telegram Only</option>
              </select>
              <button
                onClick={handleSendTest}
                disabled={sendingTest}
                className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:bg-gray-400"
              >
                {sendingTest ? 'Sending...' : 'Send Test'}
              </button>
            </div>
          </div>

          {/* Event Toggles */}
          <div>
            <h3 className="text-sm font-medium text-secondary mb-2">Notification Events</h3>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {['Quota exceeded', 'Expiry warning', 'Certificate renewed', 'Core error', 'Failed login', 'User created', 'User deleted'].map((event) => (
                <label key={event} className="flex items-center gap-2 text-sm text-primary">
                  <input type="checkbox" defaultChecked className="rounded border-primary" />
                  {event}
                </label>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Notifications List */}
      <div className="bg-surface border border-primary rounded-lg overflow-hidden">
        <div className="px-4 py-3 bg-secondary/10 border-b border-primary">
          <h3 className="font-semibold text-primary">Recent Notifications</h3>
        </div>

        {loading ? (
          <div className="p-8 text-center text-secondary">Loading...</div>
        ) : notifications.length === 0 ? (
          <div className="p-8 text-center text-secondary">No notifications yet</div>
        ) : (
          <div className="divide-y divide-primary/10">
            {notifications.map((notification) => (
              <div key={notification.id} className="p-4 hover:bg-secondary/5">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-lg">{getEventTypeIcon(notification.event_type)}</span>
                      <span className={`px-2 py-0.5 text-xs rounded-full ${getSeverityColor(notification.severity)}`}>
                        {notification.severity}
                      </span>
                      <span className="text-xs text-secondary">
                        {new Date(notification.created_at).toLocaleString()}
                      </span>
                    </div>
                    <h4 className="font-medium text-primary">{notification.title}</h4>
                    <p className="text-sm text-secondary mt-1">{notification.message}</p>
                  </div>
                  <button
                    onClick={() => handleDeleteNotification(notification.id)}
                    className="text-secondary hover:text-red-600"
                  >
                    ✕
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
