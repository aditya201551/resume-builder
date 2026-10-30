export interface Template {
  id: string
  revision: number
  renderer_key: string
  name: string
  description: string
  thumbnail_url: string | null
  tags: string[]
  supported_modes: string[]
  supported_fields: string[]
  capabilities: {
    columns: number
    headerPlacement: 'full-width' | 'main' | 'sidebar'
    photo: boolean
  }
}
