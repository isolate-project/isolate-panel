import { useState, useEffect, useRef } from 'preact/hooks'
import { backupApi, systemApi } from '../api/endpoints'
import { Modal } from '../components/ui/Modal'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Badge } from '../components/ui/Badge'
import { Card } from '../components/ui/Card'
import { Alert } from '../components/ui/Alert'
import { Spinner } from '../components/ui/Spinner'
import { Download, RefreshCw, Trash2, Calendar, Shield, Database, Globe, HardDrive } from 'lucide-preact'
import { formatBytes } from '../utils/format'
import { useToastStore } from '../stores/toastStore'

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
  const { addToast } = useToastStore()
  const [backups, setBackups] = useState<Backup[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [schedule, setSchedule] = useState<ScheduleData>({ schedule: '', next_run: '' })
  const [showScheduleForm, setShowScheduleForm] = useState(false)
  const [cronExpression, setCronExpression] = useState('')
  const [retentionCount, setRetentionCount] = useState(3)
  const [savingRetention, setSavingRetention] = useState(false)
  const [backupOptions, setBackupOptions] = useState({
    encryption_enabled: true,
    include_cores: true,
    include_certs: true,
    include_warp: true,
    include_geo: false,
  })

  // Modal State
  const [modal, setModal] = useState<{
    show: boolean
    title: string
    content: string
    onConfirm: () => void
    type: 'danger' | 'secondary' | 'primary'
    loading: boolean
  }>({
    show: false,
    title: '',
    content: '',
    onConfirm: () => {},
    type: 'primary',
    loading: false,
  })

  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    loadData()
    const interval = setInterval(loadData, 10000)
    return () => {
      mountedRef.current = false
      clearInterval(interval)
    }
  }, [])

  const loadData = async () => {
    try {
      const [backupsRes, scheduleRes, settingsRes] = await Promise.all([
        backupApi.list(),
        backupApi.getSchedule(),
        systemApi.getSettings()
      ])

      if (!mountedRef.current) return

      setBackups(backupsRes.data.data || [])
      setSchedule(scheduleRes.data.data || { schedule: '', next_run: '' })
      if (scheduleRes.data.data?.schedule) {
        setCronExpression(scheduleRes.data.data.schedule)
      }

      // Handle retention setting
      const settings = settingsRes.data.data || []
      const retentionSetting = settings.find((s: { key: string; value: string }) => s.key === 'backup_retention_count')
      if (retentionSetting) {
        setRetentionCount(parseInt(retentionSetting.value) || 3)
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
      loadData()
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to create backup: ' + (axiosErr.response?.data?.error || axiosErr.message) })
    } finally {
      setCreating(false)
    }
  }

  const handleRestoreBackup = (id: number, filename: string) => {
    setModal({
      show: true,
      title: 'Confirm System Restore',
      content: `⚠️ WARNING: Restore will completely overwrite current configuration and database!\n\nAre you sure you want to restore from "${filename}"?`,
      type: 'danger',
      loading: false,
      onConfirm: async () => {
        setModal(prev => ({ ...prev, loading: true }))
        try {
          await backupApi.restore(id, true)
          setModal(prev => ({ ...prev, show: false }))
          addToast({ type: 'success', message: 'Restore operation started! The panel will restart after completion.' })
        } catch (err: unknown) {
          const axiosErr = err as { response?: { data?: { error?: string } }; message?: string }
          addToast({ type: 'error', message: 'Failed to restore: ' + (axiosErr.response?.data?.error || axiosErr.message) })
        } finally {
          setModal(prev => ({ ...prev, loading: false }))
        }
      }
    })
  }

  const handleDeleteBackup = (id: number) => {
    setModal({
      show: true,
      title: 'Delete Backup',
      content: 'Are you sure you want to permanently delete this backup file?',
      type: 'danger',
      loading: false,
      onConfirm: async () => {
        setModal(prev => ({ ...prev, loading: true }))
        try {
          await backupApi.delete(id)
          loadData()
          setModal(prev => ({ ...prev, show: false }))
        } catch (err: unknown) {
          const axiosErr = err as { response?: { data?: { error?: string } }; message?: string }
          addToast({ type: 'error', message: 'Failed to delete: ' + (axiosErr.response?.data?.error || axiosErr.message) })
        } finally {
          setModal(prev => ({ ...prev, loading: false }))
        }
      }
    })
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
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to download: ' + (axiosErr.response?.data?.error || axiosErr.message) })
    }
  }

  const handleSetSchedule = async () => {
    try {
      await backupApi.setSchedule(cronExpression || '')
      setShowScheduleForm(false)
      loadData()
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to set schedule: ' + (axiosErr.response?.data?.error || axiosErr.message) })
    }
  }

  const handleUpdateRetention = async () => {
    setSavingRetention(true)
    try {
      await systemApi.updateSettings({
        'backup_retention_count': retentionCount.toString()
      })
      addToast({ type: 'success', message: 'Retention policy updated' })
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to update retention: ' + (axiosErr.response?.data?.error || axiosErr.message) })
    } finally {
      setSavingRetention(false)
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'completed':
        return <Badge variant="success">Completed</Badge>
      case 'running':
        return <Badge variant="info">Running</Badge>
      case 'restoring':
        return <Badge variant="warning">Restoring</Badge>
      case 'failed':
        return <Badge variant="danger">Failed</Badge>
      default:
        return <Badge variant="secondary">{status}</Badge>
    }
  }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold text-text-primary tracking-tight">Backups</h1>
          <p className="text-text-secondary mt-2">Manage system security and data disaster recovery</p>
        </div>
        <div className="flex items-center gap-3">
          <Button 
            onClick={loadData} 
            variant="ghost" 
            size="icon" 
            disabled={loading}
            title="Refresh list"
          >
            <RefreshCw className={`h-5 w-5 ${loading ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column: Schedule & Retention */}
        <div className="space-y-6">
          <Card className="overflow-hidden border-l-4 border-l-brand-primary">
            <div className="p-6 space-y-6">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-brand-primary/10 rounded-lg">
                  <Calendar className="h-5 w-5 text-brand-primary" />
                </div>
                <h3 className="font-bold text-lg text-text-primary">Automation</h3>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="flex justify-between items-center mb-2">
                    <span className="text-sm font-medium text-text-secondary">Backup Schedule</span>
                    <Badge variant={schedule.schedule ? 'info' : 'secondary'}>
                      {schedule.schedule ? 'Active' : 'Disabled'}
                    </Badge>
                  </div>
                  <div className="p-3 bg-bg-secondary rounded-xl border border-border-primary">
                    <p className="text-sm font-mono text-text-primary">
                      {schedule.schedule || 'Not scheduled'}
                    </p>
                    {schedule.next_run && (
                      <p className="text-xs text-text-muted mt-1">
                        Next: {new Date(schedule.next_run).toLocaleString()}
                      </p>
                    )}
                  </div>
                </div>

                {showScheduleForm ? (
                  <div className="space-y-3 animate-in fade-in slide-in-from-top-2 duration-200">
                    <Input
                      label="Cron Expression"
                      placeholder="0 3 * * *"
                      value={cronExpression}
                      onChange={(e) => setCronExpression((e.target as HTMLInputElement).value)}
                      helperText="Example: 0 3 * * * (Daily at 3 AM)"
                    />
                    <div className="flex gap-2">
                      <Button onClick={handleSetSchedule} className="flex-1">Save</Button>
                      <Button variant="secondary" onClick={() => setShowScheduleForm(false)}>Cancel</Button>
                    </div>
                  </div>
                ) : (
                  <Button variant="outline" className="w-full" onClick={() => setShowScheduleForm(true)}>
                    Change Schedule
                  </Button>
                )}

                <div className="pt-4 border-t border-border-primary">
                  <div className="flex justify-between items-center mb-2">
                    <span className="text-sm font-medium text-text-secondary">Retention Policy</span>
                    <Badge variant="secondary">{retentionCount} Copies</Badge>
                  </div>
                  <div className="flex gap-2">
                    <Input
                      type="number"
                      min="1"
                      max="50"
                      value={retentionCount}
                      onChange={(e) => setRetentionCount(parseInt((e.target as HTMLInputElement).value) || 1)}
                      className="w-20"
                    />
                    <Button 
                      variant="outline" 
                      onClick={handleUpdateRetention} 
                      disabled={savingRetention}
                      className="flex-1"
                    >
                      {savingRetention ? <Spinner size="sm" /> : 'Update'}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </Card>

          <Alert variant="warning">
            <div className="space-y-1">
              <strong className="block mb-1">Security Notice</strong>
              <p className="text-sm">
                Backups are encrypted using AES-256-GCM. Ensure you have a copy of the private key located at <code className="bg-yellow-100 dark:bg-yellow-900/40 px-1 rounded">/app/data/.backup_key</code>. Without it, your backups are useless.
              </p>
            </div>
          </Alert>
        </div>

        {/* Right Column: Build & List */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <div className="p-6">
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-green-500/10 rounded-lg">
                    <Database className="h-5 w-5 text-green-500" />
                  </div>
                  <h3 className="font-bold text-lg">Create Instant Backup</h3>
                </div>
              </div>

              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6 text-sm">
                <Card className={`p-3 cursor-pointer transition-all border-2 ${backupOptions.encryption_enabled ? 'border-brand-primary bg-brand-primary/5' : 'border-transparent'}`}
                      onClick={() => setBackupOptions(prev => ({ ...prev, encryption_enabled: !prev.encryption_enabled }))}>
                  <div className="flex items-center gap-2">
                    <Shield className={`h-4 w-4 ${backupOptions.encryption_enabled ? 'text-brand-primary' : 'text-text-muted'}`} />
                    <span>Encryption</span>
                  </div>
                </Card>
                <Card className={`p-3 cursor-pointer transition-all border-2 ${backupOptions.include_cores ? 'border-brand-primary bg-brand-primary/5' : 'border-transparent'}`}
                      onClick={() => setBackupOptions(prev => ({ ...prev, include_cores: !prev.include_cores }))}>
                  <div className="flex items-center gap-2">
                    <HardDrive className={`h-4 w-4 ${backupOptions.include_cores ? 'text-brand-primary' : 'text-text-muted'}`} />
                    <span>Cores</span>
                  </div>
                </Card>
                <Card className={`p-3 cursor-pointer transition-all border-2 ${backupOptions.include_certs ? 'border-brand-primary bg-brand-primary/5' : 'border-transparent'}`}
                      onClick={() => setBackupOptions(prev => ({ ...prev, include_certs: !prev.include_certs }))}>
                  <div className="flex items-center gap-2">
                    <Shield className={`h-4 w-4 ${backupOptions.include_certs ? 'text-brand-primary' : 'text-text-muted'}`} />
                    <span>Certs</span>
                  </div>
                </Card>
                <Card className={`p-3 cursor-pointer transition-all border-2 ${backupOptions.include_geo ? 'border-brand-primary bg-brand-primary/5' : 'border-transparent'}`}
                      onClick={() => setBackupOptions(prev => ({ ...prev, include_geo: !prev.include_geo }))}>
                  <div className="flex items-center gap-2">
                    <Globe className={`h-4 w-4 ${backupOptions.include_geo ? 'text-brand-primary' : 'text-text-muted'}`} />
                    <span>Geo Data</span>
                  </div>
                </Card>
              </div>

              <Button 
                onClick={handleCreateBackup} 
                className="w-full py-6 text-lg font-bold" 
                disabled={creating}
              >
                {creating ? <><Spinner className="mr-2" /> Creating...</> : 'Initiate Secure Backup'}
              </Button>
            </div>
          </Card>

          <Card className="overflow-hidden">
            <div className="p-6 border-b border-border-primary bg-bg-secondary/50">
              <h3 className="font-bold">Recent Backups</h3>
            </div>
            <div className="overflow-x-auto">
              {loading ? (
                <div className="p-12 flex flex-col items-center justify-center gap-4 text-text-muted">
                  <Spinner size="lg" />
                  <p>Querying backup repository...</p>
                </div>
              ) : backups.length === 0 ? (
                <div className="p-12 text-center text-text-muted">
                  <Database className="h-12 w-12 mx-auto mb-4 opacity-20" />
                  <p>Your backup vault is empty</p>
                </div>
              ) : (
                <table className="w-full text-left">
                  <thead className="bg-bg-secondary/50 border-b border-border-primary">
                    <tr>
                      <th className="px-6 py-4 text-xs font-semibold uppercase tracking-wider text-text-muted">Details</th>
                      <th className="px-6 py-4 text-xs font-semibold uppercase tracking-wider text-text-muted">Status</th>
                      <th className="px-6 py-4 text-xs font-semibold uppercase tracking-wider text-text-muted">Created</th>
                      <th className="px-6 py-4 text-right text-xs font-semibold uppercase tracking-wider text-text-muted">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border-primary">
                    {backups.map((backup) => (
                      <tr key={backup.id} className="hover:bg-bg-secondary/30 transition-colors">
                        <td className="px-6 py-4">
                          <div className="flex flex-col">
                            <span className="font-mono text-sm text-text-primary font-medium">{backup.filename}</span>
                            <span className="text-xs text-text-muted mt-1">
                              {formatBytes(backup.file_size_bytes)} • {backup.backup_type}
                            </span>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          {getStatusBadge(backup.status)}
                          {backup.error_message && (
                            <p className="text-[10px] text-red-500 mt-1 max-w-[150px] truncate" title={backup.error_message}>
                              {backup.error_message}
                            </p>
                          )}
                        </td>
                        <td className="px-6 py-4">
                          <span className="text-sm text-text-secondary">
                            {new Date(backup.created_at).toLocaleString()}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <div className="flex justify-end gap-1">
                            <Button 
                              variant="ghost" 
                              size="icon" 
                              onClick={() => handleDownloadBackup(backup.id, backup.filename)}
                              disabled={backup.status !== 'completed'}
                              title="Download"
                            >
                              <Download className="h-4 w-4" />
                            </Button>
                            <Button 
                              variant="ghost" 
                              size="icon" 
                              className="text-yellow-500 hover:text-yellow-600 hover:bg-yellow-500/10"
                              onClick={() => handleRestoreBackup(backup.id, backup.filename)}
                              disabled={backup.status !== 'completed'}
                              title="Restore System"
                            >
                              <RefreshCw className="h-4 w-4" />
                            </Button>
                            <Button 
                              variant="ghost" 
                              size="icon" 
                              className="text-red-500 hover:text-red-600 hover:bg-red-500/10"
                              onClick={() => handleDeleteBackup(backup.id)}
                              title="Delete permanently"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </Card>
        </div>
      </div>

      <Modal
        isOpen={modal.show}
        onClose={() => setModal(prev => ({ ...prev, show: false }))}
        title={modal.title}
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setModal(prev => ({ ...prev, show: false }))}>
              Cancel
            </Button>
            <Button 
              variant={modal.type === 'danger' ? 'danger' : 'primary'} 
              onClick={modal.onConfirm}
              disabled={modal.loading}
            >
              {modal.loading ? <Spinner size="sm" className="mr-2" /> : null}
              {modal.loading ? 'Processing...' : 'Confirm'}
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          <p className="text-text-secondary leading-relaxed whitespace-pre-wrap">{modal.content}</p>
        </div>
      </Modal>
    </div>
  )
}
