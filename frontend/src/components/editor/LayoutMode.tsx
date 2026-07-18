import { useState, type ReactNode } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { apiGet, apiPut } from '@/lib/http'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { useSectionConfigReorder } from '@/hooks/useSectionConfigReorder'
import { useAutosave } from '@/hooks/useAutosave'
import SortableList from '@/components/editor/SortableList'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { SECTION_LABELS, sectionConfigKey } from '@/lib/sectionConfig'
import type { CustomSection, FullResume, Resume, SectionConfig, Template } from '@/types/resume'

function ConfigRow({
  config,
  regions,
  customSections,
  dragHandle,
}: {
  config: SectionConfig
  regions: string[]
  customSections: CustomSection[]
  dragHandle?: ReactNode
}) {
  const { dispatch } = useResumeDraftContext()
  const [form, setForm] = useState({
    region: config.region ?? regions[0] ?? '',
    is_visible: config.is_visible,
    display_title_override: config.display_title_override ?? '',
  })

  useAutosave(form, async (value) => {
    dispatch({
      type: 'section_config_update',
      sectionType: config.section_type,
      customSectionId: config.custom_section_id,
      patch: {
        region: value.region || null,
        is_visible: value.is_visible,
        display_title_override: value.display_title_override || null,
      },
    })
  })

  const label =
    config.section_type === 'custom'
      ? customSections.find((s) => s.id === config.custom_section_id)?.title || 'Custom section'
      : SECTION_LABELS[config.section_type] ?? config.section_type

  return (
    <div className="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-2.5">
      {dragHandle}
      <Switch checked={form.is_visible} onCheckedChange={(v) => setForm((f) => ({ ...f, is_visible: v }))} />
      <span className={cn('w-40 shrink-0 text-sm', !form.is_visible && 'text-muted-foreground')}>{label}</span>
      <Input
        className="h-8 flex-1"
        placeholder="Title override"
        value={form.display_title_override}
        onChange={(e) => setForm((f) => ({ ...f, display_title_override: e.target.value }))}
      />
      <Select value={form.region} onValueChange={(v) => setForm((f) => ({ ...f, region: v }))}>
        <SelectTrigger className="h-8 w-32 shrink-0">
          <SelectValue placeholder="Region" />
        </SelectTrigger>
        <SelectContent>
          {regions.map((r) => (
            <SelectItem key={r} value={r}>
              {r}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

export default function LayoutMode({ data }: { data: FullResume }) {
  const resumeId = data.resume.id
  const { dispatch, flushNow } = useResumeDraftContext()

  const templatesQuery = useQuery({
    queryKey: ['templates'],
    queryFn: () => apiGet<Template[]>('/api/templates'),
  })

  // Template switching re-derives every section's region server-side, so
  // unlike everything else in the editor it stays an immediate network
  // call — flush any pending local edits first so the switch doesn't race
  // against (and get clobbered by) an unsynced change.
  const switchTemplate = useMutation({
    mutationFn: async (templateId: string) => {
      await flushNow()
      await apiPut<Resume>(`/api/resumes/${resumeId}/template`, { template_id: templateId })
      return apiGet<FullResume>(`/api/resumes/${resumeId}/full`)
    },
    onSuccess: (fresh) => dispatch({ type: 'replace_all', data: fresh }),
  })

  const reorder = useSectionConfigReorder(resumeId)

  const activeTemplate = templatesQuery.data?.find((t) => t.id === data.resume.template_id)
  const regions = activeTemplate?.regions ?? ['main']
  const configs = [...data.section_configs].sort((a, b) => a.sort_order - b.sort_order)

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h3 className="mb-2 text-sm font-medium text-foreground">Template</h3>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          {(templatesQuery.data ?? []).map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => switchTemplate.mutate(t.id)}
              className={cn(
                'flex flex-col items-center gap-2 rounded-md border p-3 text-xs',
                t.id === data.resume.template_id
                  ? 'border-ring ring-1 ring-ring'
                  : 'border-border hover:border-ring/60',
              )}
            >
              <div className="aspect-[3/4] w-full rounded bg-secondary" />
              {t.name}
            </button>
          ))}
        </div>
      </div>

      <div>
        <h3 className="mb-2 text-sm font-medium text-foreground">Sections</h3>
        <SortableList
          dragHandlePlacement="inline"
          items={configs.map((c) => ({ ...c, id: sectionConfigKey(c) }))}
          onReorder={(orderedKeys) => {
            const ordered = orderedKeys.map((key) => configs.find((c) => sectionConfigKey(c) === key)!)
            reorder.mutate(ordered)
          }}
          renderItem={(item, _index, dragHandle) => (
            <ConfigRow config={item} regions={regions} customSections={data.custom_sections} dragHandle={dragHandle} />
          )}
        />
      </div>
    </div>
  )
}
