import { z } from 'zod'

export const settingSchema = z.object({
  key: z.string(),
  value: z.string().optional().default(''),
  default: z.string().optional().default(''),
  env_var: z.string().optional().default(''),
  env_locked: z.boolean(),
  editable: z.boolean(),
  secret: z.boolean(),
  type: z.string(),
  options: z.array(z.string()).optional().default([]),
})

export const settingsListSchema = z.array(settingSchema)

export type Setting = z.infer<typeof settingSchema>
