import { useState, useEffect, useRef } from 'preact/hooks'
import { warpApi, coreApi } from '../api/endpoints'

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
  const [routes, setRoutes] = useState<WarpRoute[]>([])
  const [status, setStatus] = useState<WarpStatus | null>(null)
  const [presets, setPresets] = useState<Preset>({})
  const [cores, setCores] = useState<{ name: string; id: number }[]>([])
  const [selectedCore, setSelectedCore] = useState<number>(1)
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({
    resource_type: 'domain',
    resource_value: '',
    description: '',
    priority: 50,
  })

  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    loadData()
    return () => {
      abortRef.current?.abort()
    }
  }, [selectedCore])

  const loadData = async () => {
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    setLoading(true)
    try {
      const [routesRes, statusRes, presetsRes, coresRes] = await Promise.all([
        warpApi.getRoutes(selectedCore),
        warpApi.getStatus(),
        warpApi.getPresets(),
        coreApi.list(),
      ])

      if (controller.signal.aborted) return

      setRoutes(routesRes.data.data || [])
      setStatus(statusRes.data.data || null)
      setPresets(presetsRes.data.data || {})
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
    }
  }

  const handleRegister = async () => {
    if (!confirm('Register WARP device?')) return

    try {
      await warpApi.register()
      alert('WARP registered successfully!')
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to register WARP: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleApplyPreset = async (presetName: string) => {
    if (!confirm(`Apply preset "${presetName}"?`)) return

    try {
      await warpApi.applyPreset(presetName, selectedCore)
      alert('Preset applied successfully!')
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to apply preset: ' + (error.response?.data?.error || error.message))
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
      alert('Route created successfully!')
      setShowForm(false)
      setFormData({ resource_type: 'domain', resource_value: '', description: '', priority: 50 })
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to create route: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleDeleteRoute = async (id: number) => {
    if (!confirm('Delete this route?')) return

    try {
      await warpApi.deleteRoute(id)
      alert('Route deleted successfully!')
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to delete route: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleToggleRoute = async (id: number) => {
    try {
      await warpApi.toggleRoute(id)
      loadData()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to toggle route: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleSync = async () => {
    try {
      await warpApi.sync()
      alert('WARP routes synchronized!')
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      alert('Failed to sync: ' + (error.response?.data?.error || error.message))
    }
  }

  if (loading) {
    return <div class="p-4">Loading...</div>
  }

  return (
    <div class="p-6">
      <div class="mb-6">
        <h1 class="text-2xl font-bold mb-4">WARP Routes</h1>

        {/* Status Card */}
        <div class="bg-white rounded-lg shadow p-4 mb-4">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-semibold">WARP Status</h2>
            <div class="flex items-center gap-2">
              <span
                class={`px-2 py-1 rounded text-sm ${
                  status?.is_registered
                    ? 'bg-green-100 text-green-800'
                    : 'bg-gray-100 text-gray-800'
                }`}
              >
                {status?.is_registered ? 'Registered' : 'Not Registered'}
              </span>
              <span
                class={`px-2 py-1 rounded text-sm ${
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
            <div class="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span class="text-gray-600">IPv4:</span> {status.ip_address}
              </div>
              {status.ipv6_address && (
                <div>
                  <span class="text-gray-600">IPv6:</span> {status.ipv6_address}
                </div>
              )}
            </div>
          )}

          <div class="flex gap-2 mt-4">
            {!status?.is_registered && (
              <button
                onClick={handleRegister}
                class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
              >
                Register WARP
              </button>
            )}
            <button
              onClick={handleSync}
              class="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700"
            >
              Sync Routes
            </button>
          </div>
        </div>

        {/* Core Selector */}
        <div class="bg-white rounded-lg shadow p-4 mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">Select Core</label>
          <select
            value={selectedCore}
            onChange={(e) => setSelectedCore(Number(e.currentTarget.value))}
            class="w-full p-2 border rounded"
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
          <div class="bg-white rounded-lg shadow p-4 mb-4">
            <h3 class="text-lg font-semibold mb-3">Quick Presets</h3>
            <div class="flex flex-wrap gap-2">
              {Object.keys(presets).map((presetName) => (
                <button
                  key={presetName}
                  onClick={() => handleApplyPreset(presetName)}
                  class="px-3 py-1 bg-purple-100 text-purple-800 rounded hover:bg-purple-200 capitalize"
                >
                  {presetName.replace(/_/g, ' ')}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Add Route Button */}
        <div class="mb-4">
          <button
            onClick={() => setShowForm(!showForm)}
            class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
          >
            {showForm ? 'Cancel' : 'Add Route'}
          </button>
        </div>

        {/* Add Route Form */}
        {showForm && (
          <form onSubmit={handleCreateRoute} class="bg-white rounded-lg shadow p-4 mb-4">
            <div class="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Resource Type</label>
                <select
                  value={formData.resource_type}
                  onChange={(e) =>
                    setFormData({ ...formData, resource_type: e.currentTarget.value })
                  }
                  class="w-full p-2 border rounded"
                >
                  <option value="domain">Domain</option>
                  <option value="ip">IP Address</option>
                  <option value="cidr">CIDR</option>
                </select>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Priority</label>
                <input
                  type="number"
                  value={formData.priority}
                  onChange={(e) =>
                    setFormData({ ...formData, priority: Number(e.currentTarget.value) })
                  }
                  class="w-full p-2 border rounded"
                  min="1"
                  max="100"
                />
              </div>

              <div class="col-span-2">
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Resource Value
                </label>
                <input
                  type="text"
                  value={formData.resource_value}
                  onChange={(e) =>
                    setFormData({ ...formData, resource_value: e.currentTarget.value })
                  }
                  class="w-full p-2 border rounded"
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

              <div class="col-span-2">
                <label class="block text-sm font-medium text-gray-700 mb-1">Description</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.currentTarget.value })
                  }
                  class="w-full p-2 border rounded"
                  placeholder="Optional description"
                />
              </div>
            </div>

            <button
              type="submit"
              class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Create Route
            </button>
          </form>
        )}

        {/* Routes Table */}
        <div class="bg-white rounded-lg shadow overflow-hidden">
          <table class="w-full">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Type</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Value</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Priority</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Status</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody>
              {routes.length === 0 ? (
                <tr>
                  <td colSpan={5} class="px-4 py-8 text-center text-gray-500">
                    No WARP routes configured. Add a route or apply a preset.
                  </td>
                </tr>
              ) : (
                routes.map((route) => (
                  <tr key={route.id} class="border-t">
                    <td class="px-4 py-2">
                      <span class="px-2 py-1 bg-gray-100 rounded text-sm capitalize">
                        {route.resource_type}
                      </span>
                    </td>
                    <td class="px-4 py-2 font-mono text-sm">{route.resource_value}</td>
                    <td class="px-4 py-2">
                      <span class="text-sm">{route.priority}</span>
                    </td>
                    <td class="px-4 py-2">
                      <span
                        class={`px-2 py-1 rounded text-sm ${
                          route.is_enabled
                            ? 'bg-green-100 text-green-800'
                            : 'bg-gray-100 text-gray-800'
                        }`}
                      >
                        {route.is_enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td class="px-4 py-2">
                      <div class="flex gap-2">
                        <button
                          onClick={() => handleToggleRoute(route.id)}
                          class={`px-2 py-1 rounded text-sm ${
                            route.is_enabled
                              ? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200'
                              : 'bg-green-100 text-green-800 hover:bg-green-200'
                          }`}
                        >
                          {route.is_enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button
                          onClick={() => handleDeleteRoute(route.id)}
                          class="px-2 py-1 bg-red-100 text-red-800 rounded text-sm hover:bg-red-200"
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
