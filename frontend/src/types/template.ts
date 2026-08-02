import type { ResumeDesign } from './design'

// Mirrors backend/internal/repository/template_repo.go's Template struct.
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
