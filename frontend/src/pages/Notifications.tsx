import { useState, useEffect } from 'preact/hooks'
import { backupApi } from '../api/endpoints'

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
      const notificationsRes = await backupApi.list()
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
      await backupApi.delete(id)
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to delete: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleSendTest = async () => {
    setSendingTest(true)
    try {
      // Will be implemented with notificationApi
      alert('Test notification sent!')
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to send test: ' + (error.response?.data?.error || error.message))
    } finally {
      setSendingTest(false)
    }
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'bg-red-100 text-red-800'
      case 'error': return 'bg-orange-100 text-orange-800'
      case 'warning': return 'bg-yellow-100 text-yellow-800'
      case 'info': return 'bg-blue-100 text-blue-800'
      default: return 'bg-gray-100 text-gray-800'
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
    <div class="p-6">
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">Notifications</h1>
          <p class="text-gray-600 mt-1">System notifications and alerts</p>
        </div>
        <button
          onClick={() => setShowSettings(!showSettings)}
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          {showSettings ? 'Hide Settings' : 'Settings'}
        </button>
      </div>

      {/* Settings Panel */}
      {showSettings && (
        <div class="bg-white border border-gray-200 rounded-lg p-6 mb-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Notification Settings</h2>
          
          {/* Webhook Settings */}
          <div class="mb-6">
            <h3 class="text-sm font-medium text-gray-700 mb-2">Webhook</h3>
            <div class="grid grid-cols-2 gap-4">
              <input
                type="text"
                placeholder="Webhook URL"
                class="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <input
                type="password"
                placeholder="Webhook Secret"
                class="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          {/* Telegram Settings */}
          <div class="mb-6">
            <h3 class="text-sm font-medium text-gray-700 mb-2">Telegram</h3>
            <div class="grid grid-cols-2 gap-4">
              <input
                type="text"
                placeholder="Bot Token"
                class="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <input
                type="text"
                placeholder="Chat ID"
                class="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          {/* Test Notification */}
          <div class="mb-6">
            <h3 class="text-sm font-medium text-gray-700 mb-2">Test Notification</h3>
            <div class="flex gap-4">
              <select
                value={testChannel}
                onChange={(e) => setTestChannel((e.target as HTMLSelectElement).value as 'telegram' | 'webhook')}
                class="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">All Channels</option>
                <option value="webhook">Webhook Only</option>
                <option value="telegram">Telegram Only</option>
              </select>
              <button
                onClick={handleSendTest}
                disabled={sendingTest}
                class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:bg-gray-400"
              >
                {sendingTest ? 'Sending...' : 'Send Test'}
              </button>
            </div>
          </div>

          {/* Event Toggles */}
          <div>
            <h3 class="text-sm font-medium text-gray-700 mb-2">Notification Events</h3>
            <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
              {['Quota exceeded', 'Expiry warning', 'Certificate renewed', 'Core error', 'Failed login', 'User created', 'User deleted'].map((event) => (
                <label key={event} class="flex items-center gap-2 text-sm">
                  <input type="checkbox" defaultChecked class="rounded border-gray-300" />
                  {event}
                </label>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Notifications List */}
      <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div class="px-4 py-3 bg-gray-50 border-b border-gray-200">
          <h3 class="font-semibold text-gray-900">Recent Notifications</h3>
        </div>

        {loading ? (
          <div class="p-8 text-center text-gray-500">Loading...</div>
        ) : notifications.length === 0 ? (
          <div class="p-8 text-center text-gray-500">No notifications yet</div>
        ) : (
          <div class="divide-y divide-gray-200">
            {notifications.map((notification) => (
              <div key={notification.id} class="p-4 hover:bg-gray-50">
                <div class="flex items-start justify-between">
                  <div class="flex-1">
                    <div class="flex items-center gap-2 mb-1">
                      <span class="text-lg">{getEventTypeIcon(notification.event_type)}</span>
                      <span class={`px-2 py-0.5 text-xs rounded-full ${getSeverityColor(notification.severity)}`}>
                        {notification.severity}
                      </span>
                      <span class="text-xs text-gray-500">
                        {new Date(notification.created_at).toLocaleString()}
                      </span>
                    </div>
                    <h4 class="font-medium text-gray-900">{notification.title}</h4>
                    <p class="text-sm text-gray-600 mt-1">{notification.message}</p>
                  </div>
                  <button
                    onClick={() => handleDeleteNotification(notification.id)}
                    class="text-gray-400 hover:text-red-600"
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
