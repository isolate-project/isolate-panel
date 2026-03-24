import apiClient from '../client'

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
