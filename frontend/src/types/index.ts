// Shared TypeScript interfaces for API responses

export interface User {
  id: number
  uuid: string
  username: string
  email: string
  password: string
  token: string
  subscription_token: string
  is_active: boolean
  is_online: boolean
  traffic_limit_bytes: number | null
  traffic_used_bytes: number
  expire_at: string | null
  expiry_date: string | null
  created_at: string
  updated_at: string
}

export interface UsersResponse {
  users: User[]
  total: number
  page: number
  limit: number
}

export interface Core {
  id: number
  name: string
  type: string
  version: string
  is_enabled: boolean
  is_running: boolean
  pid: number | null
  uptime_seconds: number
  restart_count: number
  last_error: string
}

export interface CoreStatus {
  name: string
  is_running: boolean
  is_enabled: boolean
  pid: number | null
  uptime: number
  restarts: number
  last_error: string
}

export interface Inbound {
  id: number
  name: string
  protocol: string
  core_id: number
  listen_address: string
  port: number
  config_json: string
  tls_enabled: boolean
  tls_cert_id: number | null
  reality_enabled: boolean
  reality_config_json: string
  is_enabled: boolean
  created_at: string
  updated_at: string
  core?: {
    id: number
    name: string
    type: string
  }
}

export interface Outbound {
  id: number
  name: string
  protocol: string
  core_id: number
  config_json: string
  priority: number
  is_enabled: boolean
  created_at: string
  updated_at: string
  core?: {
    id: number
    name: string
    type: string
  }
}

export interface ProtocolSummary {
  protocol: string
  label: string
  description: string
  core: string[]
  direction: string
  requires_tls: boolean
  category: string
}

export interface ProtocolParameter {
  name: string
  label: string
  type: 'string' | 'integer' | 'boolean' | 'select' | 'uuid' | 'array' | 'object'
  required: boolean
  default?: unknown
  auto_generate: boolean
  auto_gen_func?: string
  options?: string[]
  description?: string
  example?: string
  min?: number
  max?: number
  depends_on?: ProtocolDependency[]
  group?: string
}

export interface ProtocolDependency {
  field: string
  value: unknown
  condition: string
}

export interface ProtocolSchema {
  protocol: string
  label: string
  description: string
  core: string[]
  direction: string
  requires_tls: boolean
  parameters: Record<string, ProtocolParameter>
  transport?: string[]
  category: string
}

export interface ProtocolDefaults {
  protocol: string
  defaults: Record<string, unknown>
}

export interface SystemResources {
  ram?: {
    used: number
    total: number
    percent: number
  }
  cpu?: {
    percent: number
  }
}

export interface AdminUser {
  id: number
  username: string
  is_super_admin: boolean
  created_at: string
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  expires_at: string
  admin?: AdminUser
}
