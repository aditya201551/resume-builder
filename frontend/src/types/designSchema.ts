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
