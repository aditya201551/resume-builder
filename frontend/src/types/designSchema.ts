// Mirrors backend/internal/design/schema.go's FieldDef/FieldGroup — the
// backend-served description of what's editable, so the frontend never
// hardcodes an enum list or field set that could drift from what Validate()
// actually accepts.
export type FieldType = 'enum' | 'number' | 'color' | 'bool'

export interface FieldDef {
  key: string
  label: string
  type: FieldType
  options?: string[]
  min?: number
  max?: number
  step?: number
}

export interface FieldGroup {
  key: string
  label: string
  fields: FieldDef[]
}
