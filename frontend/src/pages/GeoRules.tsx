import { useState, useEffect } from 'preact/hooks'
import { warpApi } from '../api/endpoints'

interface GeoRule {
  id: number
  core_id: number
  type: string
  code: string
  action: string
  priority: number
  is_enabled: boolean
  description: string
  created_at: string
  updated_at: string
}

interface Country {
  code: string
  name: string
}

interface Category {
  name: string
  description: string
}

export function GeoRules() {
  const [rules, setRules] = useState<GeoRule[]>([])
  const [countries, setCountries] = useState<Country[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [cores, setCores] = useState<{ name: string; id: number }[]>([])
  const [selectedCore, setSelectedCore] = useState<number>(1)
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [ruleType, setRuleType] = useState<'geoip' | 'geosite'>('geoip')
  const [formData, setFormData] = useState({
    type: 'geoip',
    code: '',
    action: 'proxy',
    priority: 50,
    description: '',
  })

  useEffect(() => {
    loadData()
  }, [selectedCore])

  const loadData = async () => {
    setLoading(true)
    try {
      const [rulesRes, countriesRes, categoriesRes] = await Promise.all([
        warpApi.getGeoRules(selectedCore),
        warpApi.getCountries(),
        warpApi.getCategories(),
      ])

      setRules(rulesRes.data.data || [])
      setCountries(countriesRes.data.data || [])
      setCategories(categoriesRes.data.data || [])

      // Load cores separately
      try {
        const coresRes = await fetch('/api/cores')
        const coresData = await coresRes.json()
        setCores((coresData.data || []).map((c: { name: string; id: number }) => ({ name: c.name, id: c.id })))
      } catch (e) {
        console.error('Failed to load cores:', e)
      }
    } catch (error) {
      console.error('Failed to load Geo data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateRule = async (e: Event) => {
    e.preventDefault()

    try {
      await warpApi.createGeoRule({
        core_id: selectedCore,
        type: formData.type,
        code: formData.code,
        action: formData.action,
        description: formData.description,
        priority: formData.priority,
      })
      alert('Geo rule created successfully!')
      setShowForm(false)
      setFormData({ type: 'geoip', code: '', action: 'proxy', priority: 50, description: '' })
      loadData()
    } catch (error: any) {
      alert('Failed to create rule: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleDeleteRule = async (id: number) => {
    if (!confirm('Delete this rule?')) return

    try {
      await warpApi.deleteGeoRule(id, selectedCore)
      alert('Rule deleted successfully!')
      loadData()
    } catch (error: any) {
      alert('Failed to delete rule: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleToggleRule = async (id: number) => {
    try {
      await warpApi.toggleGeoRule(id, selectedCore)
      loadData()
    } catch (error: any) {
      alert('Failed to toggle rule: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleUpdateDatabases = async () => {
    if (!confirm('Download/update GeoIP/GeoSite databases? This may take a minute.')) return

    try {
      await warpApi.updateDatabases()
      alert('Geo databases updated successfully!')
    } catch (error: any) {
      alert('Failed to update databases: ' + (error.response?.data?.error || error.message))
    }
  }

  if (loading) {
    return <div class="p-4">Loading...</div>
  }

  return (
    <div class="p-6">
      <div class="mb-6">
        <h1 class="text-2xl font-bold mb-4">GeoIP/GeoSite Rules</h1>

        {/* Update Databases Button */}
        <div class="bg-white rounded-lg shadow p-4 mb-4">
          <div class="flex items-center justify-between">
            <div>
              <h2 class="text-lg font-semibold">Geo Databases</h2>
              <p class="text-sm text-gray-600">
                Download and update GeoIP/GeoSite databases for routing rules
              </p>
            </div>
            <button
              onClick={handleUpdateDatabases}
              class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Update Databases
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

        {/* Add Rule Button */}
        <div class="mb-4">
          <button
            onClick={() => setShowForm(!showForm)}
            class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
          >
            {showForm ? 'Cancel' : 'Add Geo Rule'}
          </button>
        </div>

        {/* Add Rule Form */}
        {showForm && (
          <form onSubmit={handleCreateRule} class="bg-white rounded-lg shadow p-4 mb-4">
            <div class="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Rule Type</label>
                <select
                  value={formData.type}
                  onChange={(e) => {
                    const newType = e.currentTarget.value as 'geoip' | 'geosite'
                    setRuleType(newType)
                    setFormData({ ...formData, type: newType, code: '' })
                  }}
                  class="w-full p-2 border rounded"
                >
                  <option value="geoip">GeoIP (Country-based)</option>
                  <option value="geosite">GeoSite (Category-based)</option>
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
                  {ruleType === 'geoip' ? 'Country' : 'Category'}
                </label>
                <select
                  value={formData.code}
                  onChange={(e) => setFormData({ ...formData, code: e.currentTarget.value })}
                  class="w-full p-2 border rounded"
                  required
                >
                  <option value="">Select...</option>
                  {(ruleType === 'geoip' ? countries : categories).map((item: any) => (
                    <option key={item.code || item.name} value={item.code || item.name}>
                      {ruleType === 'geoip'
                        ? `${item.name} (${item.code})`
                        : `${item.name} - ${item.description}`}
                    </option>
                  ))}
                </select>
              </div>

              <div class="col-span-2">
                <label class="block text-sm font-medium text-gray-700 mb-1">Action</label>
                <select
                  value={formData.action}
                  onChange={(e) => setFormData({ ...formData, action: e.currentTarget.value })}
                  class="w-full p-2 border rounded"
                >
                  <option value="proxy">Proxy (via outbound)</option>
                  <option value="direct">Direct (bypass proxy)</option>
                  <option value="block">Block</option>
                  <option value="warp">Via WARP</option>
                </select>
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
              Create Rule
            </button>
          </form>
        )}

        {/* Rules Table */}
        <div class="bg-white rounded-lg shadow overflow-hidden">
          <table class="w-full">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Type</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Code</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Action</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Priority</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Status</th>
                <th class="px-4 py-2 text-left text-sm font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody>
              {rules.length === 0 ? (
                <tr>
                  <td colSpan={6} class="px-4 py-8 text-center text-gray-500">
                    No Geo rules configured. Add a rule to route traffic by country or category.
                  </td>
                </tr>
              ) : (
                rules.map((rule) => (
                  <tr key={rule.id} class="border-t">
                    <td class="px-4 py-2">
                      <span
                        class={`px-2 py-1 rounded text-sm ${
                          rule.type === 'geoip'
                            ? 'bg-blue-100 text-blue-800'
                            : 'bg-purple-100 text-purple-800'
                        }`}
                      >
                        {rule.type}
                      </span>
                    </td>
                    <td class="px-4 py-2 font-semibold">{rule.code.toUpperCase()}</td>
                    <td class="px-4 py-2">
                      <span
                        class={`px-2 py-1 rounded text-sm ${
                          rule.action === 'proxy'
                            ? 'bg-yellow-100 text-yellow-800'
                            : rule.action === 'direct'
                              ? 'bg-green-100 text-green-800'
                              : rule.action === 'block'
                                ? 'bg-red-100 text-red-800'
                                : 'bg-indigo-100 text-indigo-800'
                        }`}
                      >
                        {rule.action}
                      </span>
                    </td>
                    <td class="px-4 py-2">
                      <span class="text-sm">{rule.priority}</span>
                    </td>
                    <td class="px-4 py-2">
                      <span
                        class={`px-2 py-1 rounded text-sm ${
                          rule.is_enabled
                            ? 'bg-green-100 text-green-800'
                            : 'bg-gray-100 text-gray-800'
                        }`}
                      >
                        {rule.is_enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td class="px-4 py-2">
                      <div class="flex gap-2">
                        <button
                          onClick={() => handleToggleRule(rule.id)}
                          class={`px-2 py-1 rounded text-sm ${
                            rule.is_enabled
                              ? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200'
                              : 'bg-green-100 text-green-800 hover:bg-green-200'
                          }`}
                        >
                          {rule.is_enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button
                          onClick={() => handleDeleteRule(rule.id)}
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
