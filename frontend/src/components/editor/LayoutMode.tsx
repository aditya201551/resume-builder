import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPatch, apiPut } from '@/lib/http'
import { fullResumeKey } from '@/hooks/useResumeEditor'
import { useSectionConfigReorder } from '@/hooks/useSectionConfigReorder'
import { useAutosave } from '@/hooks/useAutosave'
import SortableList from '@/components/editor/SortableList'
import AutosaveStatus from '@/components/editor/AutosaveStatus'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { SECTION_LABELS, sectionConfigKey, sectionConfigUrl } from '@/lib/sectionConfig'
import type { CustomSection, FullResume, SectionConfig, Template } from '@/types/resume'

function ConfigRow({
  resumeId,
  config,
  regions,
  customSections,
}: {
  resumeId: string
  config: SectionConfig
  regions: string[]
  customSections: CustomSection[]
}) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({
    region: config.region ?? regions[0] ?? '',
    is_visible: config.is_visible,
    display_title_override: config.display_title_override ?? '',
  })

  const status = useAutosave(form, async (value) => {
    await apiPatch(sectionConfigUrl(resumeId, config), {
      region: value.region || null,
      is_visible: value.is_visible,
      display_title_override: value.display_title_override || null,
      sort_order: config.sort_order,
    })
    queryClient.invalidateQueries({ queryKey: fullResumeKey(resumeId) })
  })

  const label =
    config.section_type === 'custom'
      ? customSections.find((s) => s.id === config.custom_section_id)?.title || 'Custom section'
      : SECTION_LABELS[config.section_type] ?? config.section_type

  return (
    <div className="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-2.5">
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
      <AutosaveStatus status={status} compact className="shrink-0" />
    </div>
  )
}

export default function LayoutMode({ data }: { data: FullResume }) {
  const resumeId = data.resume.id
  const queryClient = useQueryClient()

  const templatesQuery = useQuery({
    queryKey: ['templates'],
    queryFn: () => apiGet<Template[]>('/api/templates'),
  })

  const switchTemplate = useMutation({
    mutationFn: (templateId: string) => apiPut(`/api/resumes/${resumeId}/template`, { template_id: templateId }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: fullResumeKey(resumeId) }),
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
          items={configs.map((c) => ({ ...c, id: sectionConfigKey(c) }))}
          onReorder={(orderedKeys) => {
            const ordered = orderedKeys.map((key) => configs.find((c) => sectionConfigKey(c) === key)!)
            reorder.mutate(ordered)
          }}
          renderItem={(item) => (
            <ConfigRow resumeId={resumeId} config={item} regions={regions} customSections={data.custom_sections} />
          )}
        />
      </div>
    </div>
  )
}
