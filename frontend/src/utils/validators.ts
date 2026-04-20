import { z } from 'zod'

// Zod messages are i18n key strings, resolved at display time by FormField

// User validation schema
export const userSchema = z.object({
  username: z
    .string()
    .min(3, { message: 'validation.usernameMin' })
    .max(50, { message: 'validation.usernameMax' })
    .regex(/^[a-zA-Z0-9_-]+$/, { message: 'validation.usernameFormat' }),
  email: z.string().email({ message: 'validation.emailInvalid' }).optional().or(z.literal('')),
  traffic_limit_bytes: z.number().min(0, { message: 'validation.trafficPositive' }).optional(),
  expiry_days: z.number().min(1, { message: 'validation.expiryDaysMin' }).optional(),
  unlimited: z.boolean().default(false).pipe(z.boolean()),
  is_active: z.boolean().default(true).pipe(z.boolean()),
})

export type UserFormData = z.infer<typeof userSchema>

// Inbound validation schema
export const inboundSchema = z.object({
  name: z.string().min(1, { message: 'validation.nameRequired' }).max(100, { message: 'validation.nameTooLong' }),
  protocol: z.string().min(1, { message: 'validation.protocolRequired' }),
  port: z
    .number()
    .min(1, { message: 'validation.portRange' })
    .max(65535, { message: 'validation.portRange' }),
  core_id: z.number().min(1, { message: 'validation.coreRequired' }),
  listen_address: z.string().default('0.0.0.0').pipe(z.string()),
  is_enabled: z.boolean().default(true).pipe(z.boolean()),
  tls_enabled: z.boolean().default(true).pipe(z.boolean()),
  tls_cert_id: z.number().nullable().optional(),
})

export type InboundFormData = z.infer<typeof inboundSchema>

// Login validation schema
export const loginSchema = z.object({
  username: z.string().min(1, { message: 'validation.usernameRequired' }),
  password: z.string().min(1, { message: 'validation.passwordRequired' }),
})

export type LoginFormData = z.infer<typeof loginSchema>

// Core validation schema
export const coreSchema = z.object({
  name: z.string().min(1, { message: 'validation.nameRequired' }),
  type: z.enum(['singbox', 'xray', 'mihomo']),
  version: z.string().optional(),
  config_path: z.string().optional(),
})

export type CoreFormData = z.infer<typeof coreSchema>

// Settings validation schema
export const settingsSchema = z.object({
  panel_name: z.string().min(1, { message: 'validation.panelNameRequired' }).max(100),
  jwt_access_token_ttl: z.number().min(60, { message: 'validation.accessTTLRange' }).max(86400, { message: 'validation.accessTTLRange' }),
  jwt_refresh_token_ttl: z.number().min(3600, { message: 'validation.refreshTTLRange' }).max(2592000, { message: 'validation.refreshTTLRange' }),
  max_login_attempts: z.number().min(1, { message: 'validation.loginAttemptsRange' }).max(100, { message: 'validation.loginAttemptsRange' }),
})

export type SettingsFormData = z.infer<typeof settingsSchema>
