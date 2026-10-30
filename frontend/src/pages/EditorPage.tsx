import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useLocation, useNavigate, useParams } from 'react-router'
import {
  EditorPanelContext,
  type OpenPanelHandle,
} from '@/hooks/useEditorPanel'
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
  X,
} from 'lucide-react'
import { Accordion } from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import {
  ResumeDraftProvider,
  useResumeDraftContext,
  useResumeDraftData,
} from '@/hooks/useResumeDraft'
import SectionAccordionItem from '@/components/editor/SectionAccordionItem'
import AddSectionDialog, {
  type AddSectionOption,
} from '@/components/editor/AddSectionDialog'
import ContactSection from '@/components/editor/sections/ContactSection'
import WorkExperienceSection from '@/components/editor/sections/WorkExperienceSection'
import EducationSection from '@/components/editor/sections/EducationSection'
import SkillsSection from '@/components/editor/sections/SkillsSection'
import ProjectsSection from '@/components/editor/sections/ProjectsSection'
import CertificationsSection from '@/components/editor/sections/CertificationsSection'
import LanguagesSection from '@/components/editor/sections/LanguagesSection'
import MiscEntriesSection from '@/components/editor/sections/MiscEntriesSection'
import CustomSectionsSection from '@/components/editor/sections/CustomSectionsSection'
import DesignMode from '@/components/editor/DesignMode'
import ResumePreview from '@/components/editor/ResumePreview'
import { ResumePdfProvider } from '@/hooks/useResumePdf'
import { useAgentChat } from '@/hooks/useAgentChat'
import { type AssistantTask } from '@/lib/assistantTask'
import EditorNavbar, { type EditorMode } from '@/components/editor/EditorNavbar'
import ChatPanel from '@/components/agent/ChatPanel'
import type { FullResume } from '@/types/resume'

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

const SLOT_META: Record<
  SlotKey,
  { title: string; icon: typeof Briefcase; description: string }
> = {
  work_experience: {
    title: 'Professional Experience',
    icon: Briefcase,
    description:
      'Add your professional roles and employer history, including internships.',
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
    description:
      'Add your industry certificates or licences, with issuer and date.',
  },
  languages: {
    title: 'Languages',
    icon: LanguagesIcon,
    description: 'Add the languages you speak and your proficiency level.',
  },
  awards: {
    title: 'Awards',
    icon: Trophy,
    description:
      'Add awards and recognitions from industry, competitions, or academia.',
  },
  publications: {
    title: 'Publications',
    icon: BookOpen,
    description:
      'Add publications, articles, or books you wrote or contributed to.',
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

function slotContent(
  resumeId: string,
  data: FullResume,
  key: SlotKey,
): { title: string; count: number; node: ReactNode } {
  const title = SLOT_META[key].title
  const count = slotCount(data, key)
  switch (key) {
    case 'work_experience':
      return {
        title,
        count,
        node: (
          <WorkExperienceSection
            resumeId={resumeId}
            items={data.work_experiences}
          />
        ),
      }
    case 'education':
      return {
        title,
        count,
        node: <EducationSection resumeId={resumeId} items={data.educations} />,
      }
    case 'skills':
      return {
        title,
        count,
        node: <SkillsSection resumeId={resumeId} items={data.skill_groups} />,
      }
    case 'projects':
      return {
        title,
        count,
        node: <ProjectsSection resumeId={resumeId} items={data.projects} />,
      }
    case 'certifications':
      return {
        title,
        count,
        node: (
          <CertificationsSection
            resumeId={resumeId}
            items={data.certifications}
          />
        ),
      }
    case 'languages':
      return {
        title,
        count,
        node: <LanguagesSection resumeId={resumeId} items={data.languages} />,
      }
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
      return {
        title: 'Custom Sections',
        count,
        node: (
          <CustomSectionsSection
            resumeId={resumeId}
            items={data.custom_sections}
          />
        ),
      }
  }
}

function ContentMode({
  resumeId,
  data,
  selected,
  selectionVersion,
}: {
  resumeId: string
  data: FullResume
  selected?: string
  selectionVersion?: number
}) {
  const customSections = useEntityMutations(
    resumeId,
    `/api/resumes/${resumeId}/custom-sections`,
  )
  const [revealed, setRevealed] = useState<Set<SlotKey>>(new Set())
  const [addOpen, setAddOpen] = useState(false)
  const [openSection, setOpenSection] = useState('')
  useEffect(() => {
    if (!selected) return
    setOpenSection(selected === 'summary' ? 'contact' : selected.startsWith('custom-') ? 'custom' : selected)
  }, [selected, selectionVersion])

  const isActive = (key: SlotKey) =>
    slotCount(data, key) > 0 || revealed.has(key)

  const sections = DEFAULT_SLOT_ORDER.filter(isActive).map((key) => ({
    key,
    ...slotContent(resumeId, data, key),
  }))

  const addableOptions: AddSectionOption[] = DEFAULT_SLOT_ORDER.filter(
    (key) => key !== 'custom' && !isActive(key),
  ).map((key) => ({
    key,
    title: SLOT_META[key].title,
    description: SLOT_META[key].description,
    icon: SLOT_META[key].icon,
  }))
  addableOptions.push({
    key: 'custom',
    title: SLOT_META.custom.title,
    description: SLOT_META.custom.description,
    icon: SLOT_META.custom.icon,
    emphasized: true,
  })

  function handleAdd(key: string) {
    if (key === 'custom') {
      customSections.create.mutate({
        title: '',
        sort_order: data.custom_sections.length,
      })
    } else {
      setRevealed((prev) => new Set(prev).add(key as SlotKey))
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <Accordion type="single" collapsible value={openSection === 'contact' ? 'contact' : ''} onValueChange={setOpenSection} className="flex flex-col gap-3">
        <SectionAccordionItem
          value="contact"
          title="Contact & Summary"
          count={0}
        >
          <ContactSection resume={data.resume} />
        </SectionAccordionItem>
      </Accordion>

      <Accordion type="single" collapsible value={openSection === 'contact' ? '' : openSection} onValueChange={setOpenSection} className="flex flex-col gap-3">
        {sections.map((section) => (
          <SectionAccordionItem
            key={section.key}
            value={section.key}
            title={section.title}
            count={section.count}
          >
            {section.node}
          </SectionAccordionItem>
        ))}
      </Accordion>

      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="mt-2 w-fit"
        onClick={() => setAddOpen(true)}
      >
        <Plus className="size-3.5" /> Add section
      </Button>

      <AddSectionDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        options={addableOptions}
        onSelect={handleAdd}
      />
    </div>
  )
}

function EditPane({
  resumeId,
  data,
  active,
  selected,
  selectionVersion,
}: {
  resumeId: string
  data: FullResume
  active: boolean
  selected?: string
  selectionVersion?: number
}) {
  const [panelSlot, setPanelSlot] = useState<HTMLDivElement | null>(null)
  const openStack = useRef<OpenPanelHandle[]>([])

  const pushOpen = useCallback((handle: OpenPanelHandle) => {
    openStack.current.push(handle)
    return () => {
      openStack.current = openStack.current.filter((h) => h !== handle)
    }
  }, [])

  useEffect(() => {
    if (!active) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        const top = openStack.current[openStack.current.length - 1]
        top?.close()
        return
      }
    }
    window.addEventListener('keydown', onKeyDown, true)
    return () => window.removeEventListener('keydown', onKeyDown, true)
  }, [active])

  const panelContext = useMemo(
    () => ({ slot: panelSlot, pushOpen }),
    [panelSlot, pushOpen],
  )

  return (
    <EditorPanelContext.Provider value={panelContext}>
      <div className="relative h-full flex-1 overflow-y-auto pb-10">
        <ContentMode resumeId={resumeId} data={data} selected={selected} selectionVersion={selectionVersion} />
        {/* Entry editors portal in here, replacing the list in place while it's open. */}
        <div
          ref={setPanelSlot}
          className="absolute inset-0 z-10 empty:pointer-events-none"
        />
      </div>
    </EditorPanelContext.Provider>
  )
}

function EditorPageContent() {
  const { data, isLoading } = useResumeDraftData()
  const { flushNow } = useResumeDraftContext()
  const location = useLocation()
  const navigate = useNavigate()
  const launch = useRef(
    location.state as {
      assistantTask?: AssistantTask
      mode?: EditorMode
      importWarnings?: string[]
    } | null,
  )
  const started = useRef(false)
  const [importWarnings, setImportWarnings] = useState(launch.current?.importWarnings ?? [])
  const [mode, setMode] = useState<EditorMode>(
    launch.current?.mode === 'content' ? 'content' : 'chat',
  )
  const [mobilePreview, setMobilePreview] = useState(false)
  const [selectedSection, setSelectedSection] = useState<string>()
  const [selectionVersion, setSelectionVersion] = useState(0)
  const [jobDescription, setJobDescription] = useState(
    launch.current?.assistantTask?.jobDescription ?? '',
  )
  const chat = useAgentChat()
  const { sendMessage } = chat

  useEffect(() => {
    const task = launch.current?.assistantTask
    if (
      !data ||
      started.current ||
      typeof task?.prompt !== 'string' ||
      !task.prompt.trim()
    )
      return
    // Schedule after hydration; StrictMode's initial effect cleanup cancels the first timer.
    const timer = setTimeout(() => {
      if (started.current) return
      started.current = true
      void sendMessage(task.prompt, task.jobDescription ?? '')
      void navigate(location.pathname, { replace: true, state: null })
    }, 0)
    return () => clearTimeout(timer)
  }, [data, sendMessage, navigate, location.pathname])

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (
        (e.ctrlKey || e.metaKey) &&
        !e.altKey &&
        e.key.toLowerCase() === 's'
      ) {
        e.preventDefault()
        void flushNow()
      }
    }
    window.addEventListener('keydown', onKeyDown, true)
    return () => window.removeEventListener('keydown', onKeyDown, true)
  }, [flushNow])

  if (isLoading || !data)
    return (
      <div role="status" className="p-10 text-sm text-muted-foreground">
        Opening your workspace...
      </div>
    )
  const resumeId = data.resume.id
  function changeMode(next: EditorMode) {
    setMode(next)
    setMobilePreview(false)
  }

  return (
    <ResumePdfProvider data={data}>
    <div className="workspace-shell">
      <EditorNavbar
        resume={data.resume}
        mode={mode}
        onModeChange={changeMode}
      />
      {importWarnings.length > 0 && (
        <div role="status" className="mx-4 mt-2 flex max-h-32 shrink-0 items-start gap-3 overflow-y-auto rounded-2xl border border-primary/15 bg-secondary px-4 py-3 text-xs leading-relaxed sm:mx-6">
          <div className="min-w-0 flex-1">
            <p className="font-medium">Your resume has been imported.</p>
            <ul className="mt-1 list-disc space-y-1 pl-4">
              {importWarnings.map((warning, index) => (
                <li key={index} className="break-words text-muted-foreground">{warning}</li>
              ))}
            </ul>
          </div>
          <button type="button" aria-label="Dismiss import notice" className="shrink-0 rounded-full p-1 hover:bg-background" onClick={() => setImportWarnings([])}>
            <X className="size-4" />
          </button>
        </div>
      )}
      <div
        className="mobile-view-switch"
        role="group"
        aria-label="Workspace surface"
      >
        <button
          type="button"
          aria-pressed={!mobilePreview}
          onClick={() => setMobilePreview(false)}
        >
          {mode === 'chat'
            ? 'Assistant'
            : mode === 'design'
              ? 'Design'
              : 'Edit'}
        </button>
        <button
          type="button"
          aria-pressed={mobilePreview}
          onClick={() => setMobilePreview(true)}
        >
          Preview
        </button>
      </div>
      <div
        className="workspace-content"
        data-mode={mode}
        data-mobile-preview={mobilePreview}
      >
        <div className="workspace-controls">
          <div className="workspace-pane" hidden={mode !== 'chat'}>
            <ChatPanel
              chat={chat}
              jobDescription={jobDescription}
              onJobDescriptionChange={setJobDescription}
            />
          </div>
          <div className="workspace-pane" hidden={mode !== 'content'}>
            <div className="workspace-pane-heading">
              <div>
                <h1>Make it yours.</h1>
                <p>
                  Your experience, one section at a time. Changes save
                  automatically.
                </p>
              </div>
              <button
                type="button"
                className="context-chip"
                onClick={() => changeMode('chat')}
              >
                <Sparkles className="size-3.5" /> Ask AI
              </button>
            </div>
            <div className="workspace-form">
              <EditPane
                resumeId={resumeId}
                data={data}
                active={mode === 'content' && !mobilePreview}
                selected={selectedSection}
                selectionVersion={selectionVersion}
              />
            </div>
          </div>
          <div className="workspace-pane" hidden={mode !== 'design'}>
            <div className="workspace-pane-heading">
              <div>
                <h1>A look that feels like you.</h1>
                <p>Fine-tune your template, layout, and finishing touches.</p>
              </div>
            </div>
            <div className="workspace-form">
              <DesignMode />
            </div>
          </div>
        </div>
        <ResumePreview selected={selectedSection} onSelect={(section) => { setSelectedSection(section); setSelectionVersion((n) => n + 1); changeMode('content') }} />
      </div>
    </div>
    </ResumePdfProvider>
  )
}

export default function EditorPage() {
  const { id } = useParams<{ id: string }>()

  if (!id) {
    return (
      <div className="p-10 text-sm text-muted-foreground">
        Resume not found.
      </div>
    )
  }

  return (
    <ResumeDraftProvider resumeId={id}>
      <EditorPageContent />
    </ResumeDraftProvider>
  )
}
