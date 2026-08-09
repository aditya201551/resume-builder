import type { ResumeDesign } from './design'

export interface Template {
  id: string
  renderer_key: string
  name: string
  thumbnail_url: string | null
  tags: string[]
  supported_modes: string[]
  supported_groups: string[]
  default_design: ResumeDesign
  is_premium: boolean
  published: boolean
  sort_order: number
}
