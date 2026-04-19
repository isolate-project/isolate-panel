import { useState, useEffect, useRef } from 'preact/hooks'
import { warpApi, coreApi } from '../api/endpoints'
import { useToastStore } from '../stores/toastStore'

interface WarpRoute {
  id: number
  resource_type: string
  resource_value: string
  description: string
  core_id: number
  priority: number
  is_enabled: boolean
  created_at: string
  updated_at: string
}

interface WarpStatus {
  is_registered: boolean
  is_active: boolean
  device_id?: string
  account_id?: string
  ip_address?: string
  ipv6_address?: string
}

interface Preset {
  [key: string]: { resource_type: string; resource_value: string }[]
}

export function WarpRoutes() {
  const { addToast } = useToastStore()
  const [routes, setRoutes] = useState<WarpRoute[]>([])
  const [status, setStatus] = useState<WarpStatus | null>(null)
  const [presets, setPresets] = useState<Preset>({})
  const [cores, setCores] = useState<{ name: string; id: number }[]>([])
<<<<<<< Updated upstream
  const [selectedCore, setSelectedCore] = useState<number>(1)
  const [loading, setLoading] = useState(true)
=======
  const [selectedCore, setSelectedCore] = useState<number | null>(null)
>>>>>>> Stashed changes
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({
    resource_type: 'domain',
    resource_value: '',
    description: '',
    priority: 50,
  })

  const abortRef = useRef<AbortController | null>(null)

  // Load cores list on mount (once)
  useEffect(() => {
    loadCores()
  }, [])

  // Load WARP data when selectedCore changes
  useEffect(() => {
    if (selectedCore !== null) {
      loadWarpData()
    }
    return () => {
      abortRef.current?.abort()
    }
  }, [selectedCore])

<<<<<<< Updated upstream
  const loadData = async () => {
=======
  const loadCores = async () => {
    try {
      const coresRes = await coreApi.list()
      const coresList = (coresRes.data.data || []).map((c: { name: string; id: number }) => ({
        name: c.name,
        id: c.id,
      }))
      setCores(coresList)
      if (coresList.length > 0) {
        setSelectedCore(coresList[0].id)
      }
    } catch (error) {
      console.error('Failed to load cores:', error)
      addToast({ type: 'error', message: t('warp.loadFail') || 'Failed to load cores' })
    }
  }

  const loadWarpData = async () => {
    if (selectedCore === null) return
>>>>>>> Stashed changes
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    try {
      const [routesRes, statusRes, presetsRes] = await Promise.all([
        warpApi.getRoutes(selectedCore),
        warpApi.getStatus(),
        warpApi.getPresets(),
      ])

      if (controller.signal.aborted) return

      setRoutes(routesRes.data.data || [])
      setStatus(statusRes.data.data || null)
      setPresets(presetsRes.data.data || {})
<<<<<<< Updated upstream
      setCores(
        (coresRes.data.data || []).map((c: { name: string; id: number }) => ({
          name: c.name,
          id: c.id,
        }))
      )
    } catch (error) {
      if (controller.signal.aborted) return
      console.error('Failed to load WARP data:', error)
    } finally {
      if (!controller.signal.aborted) {
        setLoading(false)
      }
=======
    } catch (error) {
      if (controller.signal.aborted) return
      console.error('Failed to load WARP data:', error)
      addToast({ type: 'error', message: t('warp.loadFail') || 'Failed to load WARP data' })
>>>>>>> Stashed changes
    }
  }

  const handleRegister = async () => {
    if (!confirm('Register WARP device?')) return

    try {
      await warpApi.register()
<<<<<<< Updated upstream
      addToast({ type: 'success', message: 'WARP registered successfully!' })
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to register WARP: ' + (error.response?.data?.error || error.message) })
=======
      addToast({ type: 'success', message: t('warp.registerSuccess') })
      loadWarpData()
    } catch (err: unknown) {
      addToast({ type: 'error', message: t('warp.registerFail') + ': ' + getApiErrorMessage(err) })
    } finally {
      setImportLoading(false)
    }
  }

  const handleImportLicense = async () => {
    if (!licenseKey.trim()) return
    setImportLoading(true)
    try {
      await warpApi.importLicense(licenseKey.trim())
      addToast({ type: 'success', message: t('warp.licenseSuccess') })
      setLicenseKey('')
      loadWarpData()
    } catch (err: unknown) {
      addToast({ type: 'error', message: t('warp.licenseFail') + ': ' + getApiErrorMessage(err) })
    } finally {
      setImportLoading(false)
    }
  }

  const handleImportConfig = async () => {
    if (!manualConfig.private_key.trim()) return
    setImportLoading(true)
    try {
      await warpApi.importConfig({
        private_key: manualConfig.private_key.trim(),
        endpoint: manualConfig.endpoint || undefined,
        ipv4: manualConfig.ipv4 || undefined,
        ipv6: manualConfig.ipv6 || undefined,
      })
      addToast({ type: 'success', message: t('warp.importSuccess') })
      setManualConfig({ private_key: '', endpoint: 'engage.cloudflareclient.com:2408', ipv4: '', ipv6: '' })
      loadWarpData()
    } catch (err: unknown) {
      addToast({ type: 'error', message: t('warp.importFail') + ': ' + getApiErrorMessage(err) })
    } finally {
      setImportLoading(false)
>>>>>>> Stashed changes
    }
  }

  const handleApplyPreset = async (presetName: string) => {
    if (!confirm(`Apply preset "${presetName}"?`)) return

    try {
      await warpApi.applyPreset(presetName, selectedCore)
<<<<<<< Updated upstream
      addToast({ type: 'success', message: 'Preset applied successfully!' })
      loadData()
=======
      addToast({ type: 'success', message: t('warp.presetSuccess') })
      loadWarpData()
>>>>>>> Stashed changes
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to apply preset: ' + (error.response?.data?.error || error.message) })
    }
  }

  const handleCreateRoute = async (e: Event) => {
    e.preventDefault()

    try {
      await warpApi.createRoute({
        core_id: selectedCore,
        resource_type: formData.resource_type,
        resource_value: formData.resource_value,
        description: formData.description,
        priority: formData.priority,
      })
      addToast({ type: 'success', message: 'Route created successfully!' })
      setShowForm(false)
      setFormData({ resource_type: 'domain', resource_value: '', description: '', priority: 50 })
      loadWarpData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to create route: ' + (error.response?.data?.error || error.message) })
    }
  }

  const handleDeleteRoute = async (id: number) => {
    if (!confirm('Delete this route?')) return

    try {
      await warpApi.deleteRoute(id)
<<<<<<< Updated upstream
      addToast({ type: 'success', message: 'Route deleted successfully!' })
      loadData()
=======
      addToast({ type: 'success', message: t('warp.routeDeleted') })
      loadWarpData()
>>>>>>> Stashed changes
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to delete route: ' + (error.response?.data?.error || error.message) })
    }
  }

  const handleToggleRoute = async (id: number) => {
    try {
      await warpApi.toggleRoute(id)
      loadWarpData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to toggle route: ' + (error.response?.data?.error || error.message) })
    }
  }

  const handleSync = async () => {
    try {
      await warpApi.sync()
      addToast({ type: 'success', message: 'WARP routes synchronized!' })
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      addToast({ type: 'error', message: 'Failed to sync: ' + (error.response?.data?.error || error.message) })
    }
  }

<<<<<<< Updated upstream
  if (loading) {
    return <div className="p-4">Loading...</div>
=======
  if (cores.length === 0) {
    return (
      <PageLayout>
        <div className="p-4">
          <h1 className="text-2xl font-bold text-primary mb-4">{t('warp.title')}</h1>
          <div className="bg-surface border border-primary rounded-lg p-4">
            <p className="text-secondary">{t('warp.noCores') || 'No proxy cores available. Please add a core first.'}</p>
          </div>
        </div>
      </PageLayout>
    )
>>>>>>> Stashed changes
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">WARP Routes</h1>

        {/* Status Card */}
        <div className="bg-white rounded-lg shadow p-4 mb-4">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">WARP Status</h2>
            <div className="flex items-center gap-2">
              <span
                className={`px-2 py-1 rounded text-sm ${
                  status?.is_registered
                    ? 'bg-green-100 text-green-800'
                    : 'bg-gray-100 text-gray-800'
                }`}
              >
                {status?.is_registered ? 'Registered' : 'Not Registered'}
              </span>
              <span
                className={`px-2 py-1 rounded text-sm ${
                  status?.is_active
                    ? 'bg-green-100 text-green-800'
                    : 'bg-red-100 text-red-800'
                }`}
              >
                {status?.is_active ? 'Active' : 'Inactive'}
              </span>
            </div>
          </div>

          {status?.ip_address && (
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-600">IPv4:</span> {status.ip_address}
              </div>
              {status.ipv6_address && (
                <div>
                  <span className="text-gray-600">IPv6:</span> {status.ipv6_address}
                </div>
              )}
            </div>
          )}

          <div className="flex gap-2 mt-4">
            {!status?.is_registered && (
              <button
                onClick={handleRegister}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
              >
                Register WARP
              </button>
            )}
            <button
              onClick={handleSync}
              className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700"
            >
              Sync Routes
            </button>
          </div>
        </div>

        {/* Core Selector */}
        <div className="bg-white rounded-lg shadow p-4 mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">Select Core</label>
          <select
            value={selectedCore}
            onChange={(e) => setSelectedCore(Number(e.currentTarget.value))}
            className="w-full p-2 border rounded"
          >
            {cores.map((core) => (
              <option key={core.id} value={core.id}>
                {core.name} (ID: {core.id})
              </option>
            ))}
          </select>
        </div>

        {/* Presets */}
        {Object.keys(presets).length > 0 && (
          <div className="bg-white rounded-lg shadow p-4 mb-4">
            <h3 className="text-lg font-semibold mb-3">Quick Presets</h3>
            <div className="flex flex-wrap gap-2">
              {Object.keys(presets).map((presetName) => (
                <button
                  key={presetName}
                  onClick={() => handleApplyPreset(presetName)}
                  className="px-3 py-1 bg-purple-100 text-purple-800 rounded hover:bg-purple-200 capitalize"
                >
                  {presetName.replace(/_/g, ' ')}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Add Route Button */}
        <div className="mb-4">
          <button
            onClick={() => setShowForm(!showForm)}
            className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
          >
            {showForm ? 'Cancel' : 'Add Route'}
          </button>
        </div>

        {/* Add Route Form */}
        {showForm && (
          <form onSubmit={handleCreateRoute} className="bg-white rounded-lg shadow p-4 mb-4">
            <div className="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Resource Type</label>
                <select
                  value={formData.resource_type}
                  onChange={(e) =>
                    setFormData({ ...formData, resource_type: e.currentTarget.value })
                  }
                  className="w-full p-2 border rounded"
                >
                  <option value="domain">Domain</option>
                  <option value="ip">IP Address</option>
                  <option value="cidr">CIDR</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Priority</label>
                <input
                  type="number"
                  value={formData.priority}
                  onChange={(e) =>
                    setFormData({ ...formData, priority: Number(e.currentTarget.value) })
                  }
                  className="w-full p-2 border rounded"
                  min="1"
                  max="100"
                />
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Resource Value
                </label>
                <input
                  type="text"
                  value={formData.resource_value}
                  onChange={(e) =>
                    setFormData({ ...formData, resource_value: e.currentTarget.value })
                  }
                  className="w-full p-2 border rounded"
                  placeholder={
                    formData.resource_type === 'domain'
                      ? 'example.com'
                      : formData.resource_type === 'ip'
                        ? '1.1.1.1'
                        : '192.168.0.0/16'
                  }
                  required
                />
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.currentTarget.value })
                  }
                  className="w-full p-2 border rounded"
                  placeholder="Optional description"
                />
              </div>
            </div>

            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Create Route
            </button>
          </form>
        )}

        {/* Routes Table */}
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left text-sm font-medium text-gray-600">Type</th>
                <th className="px-4 py-2 text-left text-sm font-medium text-gray-600">Value</th>
                <th className="px-4 py-2 text-left text-sm font-medium text-gray-600">Priority</th>
                <th className="px-4 py-2 text-left text-sm font-medium text-gray-600">Status</th>
                <th className="px-4 py-2 text-left text-sm font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody>
              {routes.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                    No WARP routes configured. Add a route or apply a preset.
                  </td>
                </tr>
              ) : (
                routes.map((route) => (
                  <tr key={route.id} className="border-t">
                    <td className="px-4 py-2">
                      <span className="px-2 py-1 bg-gray-100 rounded text-sm capitalize">
                        {route.resource_type}
                      </span>
                    </td>
                    <td className="px-4 py-2 font-mono text-sm">{route.resource_value}</td>
                    <td className="px-4 py-2">
                      <span className="text-sm">{route.priority}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span
                        className={`px-2 py-1 rounded text-sm ${
                          route.is_enabled
                            ? 'bg-green-100 text-green-800'
                            : 'bg-gray-100 text-gray-800'
                        }`}
                      >
                        {route.is_enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleToggleRoute(route.id)}
                          className={`px-2 py-1 rounded text-sm ${
                            route.is_enabled
                              ? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200'
                              : 'bg-green-100 text-green-800 hover:bg-green-200'
                          }`}
                        >
                          {route.is_enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button
                          onClick={() => handleDeleteRoute(route.id)}
                          className="px-2 py-1 bg-red-100 text-red-800 rounded text-sm hover:bg-red-200"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
