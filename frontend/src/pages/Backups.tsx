import { useState, useEffect, useRef } from 'preact/hooks'
import { backupApi } from '../api/endpoints'

interface Backup {
  id: number
  filename: string
  file_path: string
  file_size_bytes: number
  checksum_sha256: string
  backup_type: string
  destination: string
  status: string
  error_message: string
  schedule_cron: string
  encryption_enabled: boolean
  backup_source: string
  metadata: string
  duration_ms: number
  created_at: string
  completed_at: string
  updated_at: string
}

interface ScheduleData {
  schedule: string
  next_run: string
}

export function Backups() {
  const [backups, setBackups] = useState<Backup[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [schedule, setSchedule] = useState<ScheduleData>({ schedule: '', next_run: '' })
  const [showScheduleForm, setShowScheduleForm] = useState(false)
  const [cronExpression, setCronExpression] = useState('')
  const [backupOptions, setBackupOptions] = useState({
    encryption_enabled: true,
    include_cores: true,
    include_certs: true,
    include_warp: true,
    include_geo: false,
  })

  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    loadData()
    // Refresh every 10 seconds to track backup progress
    const interval = setInterval(loadData, 10000)
    return () => {
      mountedRef.current = false
      clearInterval(interval)
    }
  }, [])

  const loadData = async () => {
    try {
      const [backupsRes, scheduleRes] = await Promise.all([
        backupApi.list(),
        backupApi.getSchedule(),
      ])

      if (!mountedRef.current) return

      setBackups(backupsRes.data.data || [])
      setSchedule(scheduleRes.data.data || { schedule: '', next_run: '' })
      if (scheduleRes.data.data?.schedule) {
        setCronExpression(scheduleRes.data.data.schedule)
      }
    } catch (error) {
      if (!mountedRef.current) return
      console.error('Failed to load backups:', error)
    } finally {
      if (mountedRef.current) {
        setLoading(false)
      }
    }
  }

  const handleCreateBackup = async () => {
    setCreating(true)
    try {
      await backupApi.create({
        type: 'manual',
        ...backupOptions,
      })
      alert('Backup creation started! Check status in a few seconds.')
      loadData()
    } catch (error: any) {
      alert('Failed to create backup: ' + (error.response?.data?.error || error.message))
    } finally {
      setCreating(false)
    }
  }

  const handleRestoreBackup = async (id: number, filename: string) => {
    const confirmed = confirm(
      `⚠️ WARNING: Restore will overwrite all data!\n\nRestore from ${filename}?`
    )
    if (!confirmed) return

    try {
      await backupApi.restore(id, true)
      alert('Restore operation started! The panel will restart after completion.')
    } catch (error: any) {
      alert('Failed to restore: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleDeleteBackup = async (id: number) => {
    if (!confirm('Delete this backup?')) return

    try {
      await backupApi.delete(id)
      alert('Backup deleted')
      loadData()
    } catch (error: any) {
      alert('Failed to delete: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleDownloadBackup = async (id: number, filename: string) => {
    try {
      const response = await backupApi.download(id)
      const blob = new Blob([response.data])
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch (error: any) {
      alert('Failed to download: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleSetSchedule = async () => {
    try {
      await backupApi.setSchedule(cronExpression || '')
      alert('Schedule updated')
      setShowScheduleForm(false)
      loadData()
    } catch (error: any) {
      alert('Failed to set schedule: ' + (error.response?.data?.error || error.message))
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'bg-green-100 text-green-800'
      case 'running':
        return 'bg-blue-100 text-blue-800'
      case 'failed':
        return 'bg-red-100 text-red-800'
      case 'restoring':
        return 'bg-yellow-100 text-yellow-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getCronDescription = (cron: string) => {
    if (!cron) return 'Not scheduled'
    // Simple cron parser for common patterns
    if (cron === '0 0 * * *') return 'Daily at midnight'
    if (cron === '0 3 * * *') return 'Daily at 3 AM'
    if (cron === '0 0 * * 0') return 'Weekly on Sunday'
    if (cron === '0 */6 * * *') return 'Every 6 hours'
    return cron
  }

  return (
    <div class="p-6">
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-gray-900">Backups</h1>
        <p class="text-gray-600 mt-1">Manage system backups and restore points</p>
      </div>

      {/* Schedule Info */}
      <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="font-semibold text-blue-900">Backup Schedule</h3>
            <p class="text-sm text-blue-700 mt-1">
              {getCronDescription(schedule.schedule)}
              {schedule.next_run && (
                <span class="ml-2">
                  • Next run: {new Date(schedule.next_run).toLocaleString()}
                </span>
              )}
            </p>
          </div>
          <button
            onClick={() => setShowScheduleForm(!showScheduleForm)}
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            {showScheduleForm ? 'Cancel' : 'Edit Schedule'}
          </button>
        </div>

        {showScheduleForm && (
          <div class="mt-4 pt-4 border-t border-blue-200">
            <label class="block text-sm font-medium text-blue-900 mb-2">
              Cron Expression (e.g., "0 3 * * *" for daily at 3 AM)
            </label>
            <div class="flex gap-2">
              <input
                type="text"
                value={cronExpression}
                onChange={(e) => setCronExpression((e.target as HTMLInputElement).value)}
                placeholder="0 3 * * *"
                class="flex-1 px-3 py-2 border border-blue-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={handleSetSchedule}
                class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
              >
                Save
              </button>
              {cronExpression && (
                <button
                  onClick={() => {
                    setCronExpression('')
                    handleSetSchedule()
                  }}
                  class="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700"
                >
                  Clear
                </button>
              )}
            </div>
            <p class="text-xs text-blue-600 mt-2">
              Examples: "0 0 * * *" (daily midnight), "0 0 * * 0" (weekly Sunday), "0 */6 * * *" (every 6 hours)
            </p>
          </div>
        )}
      </div>

      {/* Create Backup Button */}
      <div class="bg-gray-50 border border-gray-200 rounded-lg p-4 mb-6">
        <h3 class="font-semibold text-gray-900 mb-3">Create New Backup</h3>
        
        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3 mb-4">
          <label class="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={backupOptions.encryption_enabled}
              onChange={(e) => setBackupOptions({ ...backupOptions, encryption_enabled: (e.target as HTMLInputElement).checked })}
              class="rounded border-gray-300"
            />
            Encrypt
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={backupOptions.include_cores}
              onChange={(e) => setBackupOptions({ ...backupOptions, include_cores: (e.target as HTMLInputElement).checked })}
              class="rounded border-gray-300"
            />
            Cores
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={backupOptions.include_certs}
              onChange={(e) => setBackupOptions({ ...backupOptions, include_certs: (e.target as HTMLInputElement).checked })}
              class="rounded border-gray-300"
            />
            Certs
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={backupOptions.include_warp}
              onChange={(e) => setBackupOptions({ ...backupOptions, include_warp: (e.target as HTMLInputElement).checked })}
              class="rounded border-gray-300"
            />
            WARP
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={backupOptions.include_geo}
              onChange={(e) => setBackupOptions({ ...backupOptions, include_geo: (e.target as HTMLInputElement).checked })}
              class="rounded border-gray-300"
            />
            Geo
          </label>
        </div>

        <button
          onClick={handleCreateBackup}
          disabled={creating}
          class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:bg-gray-400"
        >
          {creating ? 'Creating...' : 'Create Backup'}
        </button>
      </div>

      {/* Backups List */}
      <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div class="px-4 py-3 bg-gray-50 border-b border-gray-200">
          <h3 class="font-semibold text-gray-900">Backup History</h3>
        </div>

        {loading ? (
          <div class="p-8 text-center text-gray-500">Loading...</div>
        ) : backups.length === 0 ? (
          <div class="p-8 text-center text-gray-500">No backups yet</div>
        ) : (
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead class="bg-gray-50">
                <tr>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Filename</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Size</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Type</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody>
                {backups.map((backup) => (
                  <tr key={backup.id} class="border-t border-gray-200 hover:bg-gray-50">
                    <td class="px-4 py-3 text-sm text-gray-900">#{backup.id}</td>
                    <td class="px-4 py-3 text-sm text-gray-900 font-mono">{backup.filename}</td>
                    <td class="px-4 py-3 text-sm text-gray-600">
                      {formatBytes(backup.file_size_bytes)}
                    </td>
                    <td class="px-4 py-3 text-sm text-gray-600 capitalize">{backup.backup_type}</td>
                    <td class="px-4 py-3">
                      <span class={`px-2 py-1 text-xs rounded-full ${getStatusColor(backup.status)}`}>
                        {backup.status}
                      </span>
                      {backup.error_message && (
                        <p class="text-xs text-red-600 mt-1">{backup.error_message}</p>
                      )}
                    </td>
                    <td class="px-4 py-3 text-sm text-gray-600">
                      {new Date(backup.created_at).toLocaleString()}
                    </td>
                    <td class="px-4 py-3 text-sm">
                      <div class="flex gap-2">
                        <button
                          onClick={() => handleDownloadBackup(backup.id, backup.filename)}
                          disabled={backup.status !== 'completed'}
                          class="text-blue-600 hover:text-blue-800 disabled:text-gray-400"
                          title="Download"
                        >
                          Download
                        </button>
                        <button
                          onClick={() => handleRestoreBackup(backup.id, backup.filename)}
                          disabled={backup.status !== 'completed'}
                          class="text-yellow-600 hover:text-yellow-800 disabled:text-gray-400"
                          title="Restore"
                        >
                          Restore
                        </button>
                        <button
                          onClick={() => handleDeleteBackup(backup.id)}
                          class="text-red-600 hover:text-red-800"
                          title="Delete"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Info Box */}
      <div class="mt-6 bg-yellow-50 border border-yellow-200 rounded-lg p-4">
        <h4 class="font-semibold text-yellow-900 mb-2">Important Notes</h4>
        <ul class="text-sm text-yellow-800 space-y-1">
          <li>• Backups include database, core configurations, certificates, and WARP keys</li>
          <li>• Encryption uses AES-256-GCM with a key stored in /app/data/.backup_key</li>
          <li>• Keep the encryption key safe - without it, backups cannot be restored</li>
          <li>• Only 3 most recent backups are kept (automatic rotation)</li>
          <li>• Restore operation will completely overwrite the current system state</li>
        </ul>
      </div>
    </div>
  )
}
