import { useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import {
  Award,
  Briefcase,
  BookOpen,
  FolderGit2,
  GraduationCap,
  HeartHandshake,
  Languages as LanguagesIcon,
  Plus,
  Puzzle,
  Sparkles,
  Trophy,
} from 'lucide-react'
import { Accordion } from '@/components/ui/accordion'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import { useSectionConfigReorder } from '@/hooks/useSectionConfigReorder'
import { ResumeDraftProvider, useResumeDraftContext, useResumeDraftData } from '@/hooks/useResumeDraft'
import { isDraftDirty } from '@/lib/resumeDraftSync'
import { cn } from '@/lib/utils'
import SortableList from '@/components/editor/SortableList'
import SectionAccordionItem from '@/components/editor/SectionAccordionItem'
import AddSectionDialog, { type AddSectionOption } from '@/components/editor/AddSectionDialog'
import ContactSection from '@/components/editor/sections/ContactSection'
import WorkExperienceSection from '@/components/editor/sections/WorkExperienceSection'
import EducationSection from '@/components/editor/sections/EducationSection'
import SkillsSection from '@/components/editor/sections/SkillsSection'
import ProjectsSection from '@/components/editor/sections/ProjectsSection'
import CertificationsSection from '@/components/editor/sections/CertificationsSection'
import LanguagesSection from '@/components/editor/sections/LanguagesSection'
import MiscEntriesSection from '@/components/editor/sections/MiscEntriesSection'
import CustomSectionsSection from '@/components/editor/sections/CustomSectionsSection'
import LayoutMode from '@/components/editor/LayoutMode'
import LivePreview from '@/components/editor/LivePreview'
import type { FullResume, SectionConfig } from '@/types/resume'

function GlobalSyncStatus() {
  const { state, syncStatus } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced)
  const label =
    syncStatus === 'syncing'
      ? 'Saving…'
      : syncStatus === 'error'
        ? "Couldn't save — retrying"
        : dirty
          ? 'Unsaved changes'
          : 'All changes saved'
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 text-xs',
        syncStatus === 'error' ? 'text-destructive' : 'text-muted-foreground',
      )}
    >
      <span
        className={cn(
          'size-1.5 rounded-full',
          syncStatus === 'syncing' && 'animate-pulse bg-accent',
          syncStatus === 'error' && 'bg-destructive',
          syncStatus === 'idle' && (dirty ? 'bg-muted-foreground/50' : 'bg-accent'),
        )}
      />
      {label}
    </span>
  )
}

const PINNED_TYPES = new Set(['contact', 'summary'])
const DEFAULT_SLOT_ORDER = [
  'work_experience',
  'education',
  'skills',
  'projects',
  'certifications',
  'languages',
  'awards',
  'publications',
  'volunteer',
  'custom',
] as const

type SlotKey = (typeof DEFAULT_SLOT_ORDER)[number]

const SLOT_META: Record<SlotKey, { title: string; icon: typeof Briefcase; description: string }> = {
  work_experience: {
    title: 'Professional Experience',
    icon: Briefcase,
    description: 'Add your professional roles and employer history, including internships.',
  },
  education: {
    title: 'Education',
    icon: GraduationCap,
    description: 'Add your degrees and schools, honors, or exchange terms.',
  },
  skills: {
    title: 'Skills',
    icon: Sparkles,
    description: 'Add the hard and soft skills that help you stand out.',
  },
  projects: {
    title: 'Projects',
    icon: FolderGit2,
    description: 'Add key projects you contributed to and the impact they had.',
  },
  certifications: {
    title: 'Certificates',
    icon: Award,
    description: 'Add your industry certificates or licences, with issuer and date.',
  },
  languages: {
    title: 'Languages',
    icon: LanguagesIcon,
    description: 'Add the languages you speak and your proficiency level.',
  },
  awards: {
    title: 'Awards',
    icon: Trophy,
    description: 'Add awards and recognitions from industry, competitions, or academia.',
  },
  publications: {
    title: 'Publications',
    icon: BookOpen,
    description: 'Add publications, articles, or books you wrote or contributed to.',
  },
  volunteer: {
    title: 'Volunteer',
    icon: HeartHandshake,
    description: 'Add memberships or volunteering, including your role.',
  },
  custom: {
    title: 'Custom Section',
    icon: Puzzle,
    description: 'Add a section for anything else, in whatever shape you need.',
  },
}

function slotCount(data: FullResume, key: SlotKey): number {
  switch (key) {
    case 'work_experience':
      return data.work_experiences.length
    case 'education':
      return data.educations.length
    case 'skills':
      return data.skill_groups.length
    case 'projects':
      return data.projects.length
    case 'certifications':
      return data.certifications.length
    case 'languages':
      return data.languages.length
    case 'awards':
      return data.misc_entries.filter((e) => e.kind === 'award').length
    case 'publications':
      return data.misc_entries.filter((e) => e.kind === 'publication').length
    case 'volunteer':
      return data.misc_entries.filter((e) => e.kind === 'volunteer').length
    case 'custom':
      return data.custom_sections.length
  }
}

function slotContent(resumeId: string, data: FullResume, key: SlotKey): { title: string; count: number; node: ReactNode } {
  const title = SLOT_META[key].title
  const count = slotCount(data, key)
  switch (key) {
    case 'work_experience':
      return { title, count, node: <WorkExperienceSection resumeId={resumeId} items={data.work_experiences} /> }
    case 'education':
      return { title, count, node: <EducationSection resumeId={resumeId} items={data.educations} /> }
    case 'skills':
      return { title, count, node: <SkillsSection resumeId={resumeId} items={data.skill_groups} /> }
    case 'projects':
      return { title, count, node: <ProjectsSection resumeId={resumeId} items={data.projects} /> }
    case 'certifications':
      return { title, count, node: <CertificationsSection resumeId={resumeId} items={data.certifications} /> }
    case 'languages':
      return { title, count, node: <LanguagesSection resumeId={resumeId} items={data.languages} /> }
    case 'awards':
      return {
        title,
        count,
        node: (
          <MiscEntriesSection
            resumeId={resumeId}
            kind="award"
            items={data.misc_entries}
            icon={Trophy}
            emptyMessage="No awards added yet."
            addLabel="Add award"
          />
        ),
      }
    case 'publications':
      return {
        title,
        count,
        node: (
          <MiscEntriesSection
            resumeId={resumeId}
            kind="publication"
            items={data.misc_entries}
            icon={BookOpen}
            emptyMessage="No publications added yet."
            addLabel="Add publication"
          />
        ),
      }
    case 'volunteer':
      return {
        title,
        count,
        node: (
          <MiscEntriesSection
            resumeId={resumeId}
            kind="volunteer"
            items={data.misc_entries}
            icon={HeartHandshake}
            emptyMessage="No volunteer entries added yet."
            addLabel="Add volunteer entry"
          />
        ),
      }
    case 'custom':
      return { title: 'Custom Sections', count, node: <CustomSectionsSection resumeId={resumeId} items={data.custom_sections} /> }
  }
}

/**
 * Content mode's section order is the same `resume_section_configs.sort_order`
 * Layout mode edits — Contact & Summary stay pinned first (they're the
 * identity block, always index 0/1); everything else is drag-orderable and
 * writes straight back through the shared reorder mutation.
 */
function useContentSlotOrder(data: FullResume) {
  const reorderable = data.section_configs.filter((c) => !PINNED_TYPES.has(c.section_type))
  const nonCustom = reorderable.filter((c) => c.section_type !== 'custom').sort((a, b) => a.sort_order - b.sort_order)
  const customRows = reorderable.filter((c) => c.section_type === 'custom').sort((a, b) => a.sort_order - b.sort_order)

  let slotKeys: SlotKey[]
  if (reorderable.length > 0) {
    const customSortOrder = customRows[0]?.sort_order ?? Infinity
    const slots: { key: SlotKey; sortOrder: number }[] = nonCustom.map((c) => ({
      key: c.section_type as SlotKey,
      sortOrder: c.sort_order,
    }))
    if (data.custom_sections.length > 0) slots.push({ key: 'custom', sortOrder: customSortOrder })
    slotKeys = slots.sort((a, b) => a.sortOrder - b.sortOrder).map((s) => s.key)
  } else {
    slotKeys = [...DEFAULT_SLOT_ORDER]
  }

  function buildFlatOrder(newSlotKeys: string[]): SectionConfig[] {
    const pinned = data.section_configs.filter((c) => PINNED_TYPES.has(c.section_type)).sort((a, b) => a.sort_order - b.sort_order)
    const flat: SectionConfig[] = []
    newSlotKeys.forEach((key) => {
      if (key === 'custom') {
        flat.push(...customRows)
      } else {
        const row = nonCustom.find((c) => c.section_type === key)
        if (row) flat.push(row)
      }
    })
    return [...pinned, ...flat]
  }

  return { slotKeys, canPersist: reorderable.length > 0, buildFlatOrder }
}

function ContentMode({ resumeId, data }: { resumeId: string; data: FullResume }) {
  const reorder = useSectionConfigReorder(resumeId)
  const { slotKeys, canPersist, buildFlatOrder } = useContentSlotOrder(data)
  const customSections = useEntityMutations(resumeId, `/api/resumes/${resumeId}/custom-sections`)
  const [revealed, setRevealed] = useState<Set<SlotKey>>(new Set())
  const [addOpen, setAddOpen] = useState(false)

  const isActive = (key: SlotKey) => slotCount(data, key) > 0 || revealed.has(key)

  const sections = slotKeys.filter(isActive).map((key) => ({ key, ...slotContent(resumeId, data, key) }))

  const addableOptions: AddSectionOption[] = DEFAULT_SLOT_ORDER.filter((key) => key !== 'custom' && !isActive(key)).map(
    (key) => ({ key, title: SLOT_META[key].title, description: SLOT_META[key].description, icon: SLOT_META[key].icon }),
  )
  addableOptions.push({
    key: 'custom',
    title: SLOT_META.custom.title,
    description: SLOT_META.custom.description,
    icon: SLOT_META.custom.icon,
    emphasized: true,
  })

  function handleAdd(key: string) {
    if (key === 'custom') {
      customSections.create.mutate({ title: '', sort_order: data.custom_sections.length })
    } else {
      setRevealed((prev) => new Set(prev).add(key as SlotKey))
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <Accordion type="single" collapsible defaultValue="contact" className="flex flex-col gap-2">
        <SectionAccordionItem value="contact" title="Contact & Summary" count={0}>
          <ContactSection resume={data.resume} />
        </SectionAccordionItem>
      </Accordion>

      <Accordion type="single" collapsible className="flex flex-col gap-2">
        <SortableList
          items={sections.map((s) => ({ id: s.key }))}
          onReorder={(orderedKeys) => {
            if (!canPersist) return
            reorder.mutate(buildFlatOrder(orderedKeys))
          }}
          renderItem={(item) => {
            const section = sections.find((s) => s.key === item.id)!
            return (
              <SectionAccordionItem value={section.key} title={section.title} count={section.count}>
                {section.node}
              </SectionAccordionItem>
            )
          }}
        />
      </Accordion>

      <Button type="button" variant="secondary" size="sm" className="mt-2 w-fit" onClick={() => setAddOpen(true)}>
        <Plus className="size-3.5" /> Add section
      </Button>

      <AddSectionDialog open={addOpen} onOpenChange={setAddOpen} options={addableOptions} onSelect={handleAdd} />
    </div>
  )
}

function EditorPageContent() {
  const { data, isLoading } = useResumeDraftData()

  if (isLoading || !data) {
    return <div className="p-10 text-sm text-muted-foreground">Loading resume…</div>
  }

  const resumeId = data.resume.id

  const editPane = (
    <Tabs defaultValue="content" className="flex h-full flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/resumes">← Dashboard</Link>
        </Button>
        <TabsList>
          <TabsTrigger value="content">Content</TabsTrigger>
          <TabsTrigger value="layout">Layout</TabsTrigger>
        </TabsList>
        <div className="ml-auto">
          <GlobalSyncStatus />
        </div>
      </div>
      <TabsContent value="content" className="flex-1 overflow-y-auto pb-10">
        <ContentMode resumeId={resumeId} data={data} />
      </TabsContent>
      <TabsContent value="layout" className="flex-1 overflow-y-auto pb-10">
        <LayoutMode data={data} />
      </TabsContent>
    </Tabs>
  )

  const previewPane = (
    <div className="h-full overflow-y-auto bg-secondary/40 p-6">
      <LivePreview data={data} />
    </div>
  )

  return (
    <>
      {/* Desktop split pane */}
      <div className="hidden min-[900px]:grid min-[900px]:h-svh min-[900px]:grid-cols-2">
        <div className="overflow-y-auto px-6 py-6">{editPane}</div>
        {previewPane}
      </div>

      {/* Mobile: Edit/Preview tab switcher */}
      <div className="min-[900px]:hidden">
        <Tabs defaultValue="edit">
          <div className="sticky top-0 z-10 flex justify-center border-b border-border bg-background py-2">
            <TabsList>
              <TabsTrigger value="edit">Edit</TabsTrigger>
              <TabsTrigger value="preview">Preview</TabsTrigger>
            </TabsList>
          </div>
          <TabsContent value="edit" className="px-4 py-4">
            {editPane}
          </TabsContent>
          <TabsContent value="preview" className="bg-secondary/40 p-4">
            <LivePreview data={data} />
          </TabsContent>
        </Tabs>
      </div>
    </>
  )
}

export default function EditorPage() {
  const { id } = useParams<{ id: string }>()

  if (!id) {
    return <div className="p-10 text-sm text-muted-foreground">Resume not found.</div>
  }

  return (
    <ResumeDraftProvider resumeId={id}>
      <EditorPageContent />
    </ResumeDraftProvider>
  )
}
