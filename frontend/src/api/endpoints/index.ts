import apiClient from '../client'
export { apiClient }

// Auth endpoints
export const authApi = {
  login: (username: string, password: string) =>
    apiClient.post('/auth/login', { username, password }),

  refresh: (refreshToken: string) =>
    apiClient.post('/auth/refresh', { refresh_token: refreshToken }),

  logout: (refreshToken: string) =>
    apiClient.post('/auth/logout', { refresh_token: refreshToken }),

  me: () => apiClient.get('/me'),
}

// User endpoints
export const userApi = {
  list: (params?: { page?: number; limit?: number }) =>
    apiClient.get('/users', { params }),

  get: (id: number) => apiClient.get(`/users/${id}`),

  create: (data: Record<string, unknown>) => apiClient.post('/users', data),

  update: (id: number, data: Record<string, unknown>) => apiClient.put(`/users/${id}`, data),

  delete: (id: number) => apiClient.delete(`/users/${id}`),

  regenerate: (id: number) => apiClient.post(`/users/${id}/regenerate`),

  getInbounds: (id: number) => apiClient.get(`/users/${id}/inbounds`),
}

// Core endpoints
export const coreApi = {
  list: () => apiClient.get('/cores'),

  get: (name: string) => apiClient.get(`/cores/${name}`),

  start: (name: string) => apiClient.post(`/cores/${name}/start`),

  stop: (name: string) => apiClient.post(`/cores/${name}/stop`),

  restart: (name: string) => apiClient.post(`/cores/${name}/restart`),

  status: (name: string) => apiClient.get(`/cores/${name}/status`),

  logs: (name: string, params?: { lines?: number; since?: string }) =>
    apiClient.get(`/cores/${name}/logs`, { params }),
}

// Inbound endpoints
export const inboundApi = {
  list: () => apiClient.get('/inbounds'),

  get: (id: number) => apiClient.get(`/inbounds/${id}`),

  create: (data: Record<string, unknown>) => apiClient.post('/inbounds', data),

  update: (id: number, data: Record<string, unknown>) => apiClient.put(`/inbounds/${id}`, data),

  delete: (id: number) => apiClient.delete(`/inbounds/${id}`),

  getByCore: (coreId: number) => apiClient.get(`/inbounds/core/${coreId}`),

  assign: (inboundId: number, userId: number) =>
    apiClient.post('/inbounds/assign', { inbound_id: inboundId, user_id: userId }),

  unassign: (inboundId: number, userId: number) =>
    apiClient.post('/inbounds/unassign', { inbound_id: inboundId, user_id: userId }),

  getUsers: (id: number) => apiClient.get(`/inbounds/${id}/users`),

  bulkAssignUsers: (id: number, addUserIds: number[], removeUserIds: number[]) =>
    apiClient.post(`/inbounds/${id}/users/bulk`, { add_user_ids: addUserIds, remove_user_ids: removeUserIds }),
}

// Outbound endpoints
export const outboundApi = {
  list: (params?: { core_id?: number; protocol?: string }) =>
    apiClient.get('/outbounds', { params }),

  get: (id: number) => apiClient.get(`/outbounds/${id}`),

  create: (data: Record<string, unknown>) => apiClient.post('/outbounds', data),

  update: (id: number, data: Record<string, unknown>) => apiClient.put(`/outbounds/${id}`, data),

  delete: (id: number) => apiClient.delete(`/outbounds/${id}`),
}

// Protocol endpoints
export const protocolApi = {
  list: (params?: { core?: string; direction?: string }) =>
    apiClient.get('/protocols', { params }),

  get: (name: string) => apiClient.get(`/protocols/${name}`),

  getDefaults: (name: string) => apiClient.get(`/protocols/${name}/defaults`),
}

// Subscription endpoints (admin)
export const subscriptionApi = {
  getShortURL: (userId: number, token: string) =>
    apiClient.get(`/subscriptions/${userId}/short-url`, { params: { token } }),

  getStats: (userId: number, days?: number) =>
    apiClient.get(`/users/${userId}/subscription/stats`, { params: days ? { days } : {} }),

  regenerateToken: (userId: number) =>
    apiClient.post(`/users/${userId}/subscription/regenerate`),
}

// Certificate endpoints
export const certificateApi = {
  list: () => apiClient.get('/certificates'),
  dropdown: () => apiClient.get('/certificates/dropdown'),
  request: (data: { domain: string; is_wildcard: boolean }) => apiClient.post('/certificates', data),
  upload: (data: { certificate: string; private_key: string; issuer?: string; domain: string; is_wildcard: boolean }) =>
    apiClient.post('/certificates/upload', data),
  get: (id: number) => apiClient.get(`/certificates/${id}`),
  renew: (id: number) => apiClient.post(`/certificates/${id}/renew`),
  revoke: (id: number) => apiClient.post(`/certificates/${id}/revoke`),
  delete: (id: number) => apiClient.delete(`/certificates/${id}`),
}

// Stats and monitoring endpoints
export const statsApi = {
  dashboard: () => apiClient.get('/stats/dashboard'),
  userTraffic: (userId: number, params?: { granularity?: string; days?: number }) =>
    apiClient.get(`/stats/user/${userId}/traffic`, { params }),
  connections: (userId?: number) =>
    apiClient.get('/stats/connections', { params: userId ? { user_id: userId } : {} }),
  disconnectUser: (userId: number) =>
    apiClient.post(`/stats/user/${userId}/disconnect`),
}

// System endpoints
export const systemApi = {
  health: () => apiClient.get('/health'),

  resources: () => apiClient.get('/system/resources'),

  emergencyCleanup: () => apiClient.post('/system/emergency-cleanup'),

  getSettings: () => apiClient.get('/settings'),

  updateSettings: (data: Record<string, unknown>) => apiClient.put('/settings', data),

  connections: () => apiClient.get('/system/connections'),
}

// WARP endpoints
export const warpApi = {
  // WARP Routes
  getRoutes: (coreId: number) => apiClient.get('/warp/routes', { params: { core_id: coreId } }),

  createRoute: (data: {
    core_id: number
    resource_type: string
    resource_value: string
    description?: string
    priority?: number
  }) => apiClient.post('/warp/routes', data),

  updateRoute: (id: number, data: Record<string, unknown>) =>
    apiClient.put(`/warp/routes/${id}`, data),

  deleteRoute: (id: number) => apiClient.delete(`/warp/routes/${id}`),

  toggleRoute: (id: number) => apiClient.post(`/warp/routes/${id}/toggle`),

  sync: () => apiClient.post('/warp/sync'),

  // WARP Status & Registration
  getStatus: () => apiClient.get('/warp/status'),

  register: () => apiClient.post('/warp/register'),

  // WARP Presets
  getPresets: () => apiClient.get('/warp/presets'),

  applyPreset: (presetName: string, coreId: number) =>
    apiClient.post(`/warp/presets/${presetName}/apply`, null, { params: { core_id: coreId } }),

  // Geo Rules
  getGeoRules: (coreId: number) =>
    apiClient.get('/geo/rules', { params: { core_id: coreId } }),

  createGeoRule: (data: {
    core_id: number
    type: string
    code: string
    action: string
    description?: string
    priority?: number
  }) => apiClient.post('/geo/rules', data),

  updateGeoRule: (id: number, data: Record<string, unknown>) =>
    apiClient.put(`/geo/rules/${id}`, data),

  deleteGeoRule: (id: number, coreId: number) =>
    apiClient.delete(`/geo/rules/${id}`, { params: { core_id: coreId } }),

  toggleGeoRule: (id: number, coreId: number) =>
    apiClient.post(`/geo/rules/${id}/toggle`, null, { params: { core_id: coreId } }),

  // Geo Data
  getCountries: () => apiClient.get('/geo/countries'),

  getCategories: () => apiClient.get('/geo/categories'),

  getDatabases: () => apiClient.get('/geo/databases'),

  updateDatabases: () => apiClient.post('/geo/update'),
}
