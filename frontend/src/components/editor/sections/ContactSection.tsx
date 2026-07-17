import { useState } from 'react'
import { Plus, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { useAutosave } from '@/hooks/useAutosave'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import type { Resume, ResumeLink } from '@/types/resume'

export default function ContactSection({ resume }: { resume: Resume }) {
  const { dispatch } = useResumeDraftContext()
  const [form, setForm] = useState({
    full_name: resume.full_name,
    headline: resume.headline ?? '',
    email: resume.email ?? '',
    phone: resume.phone ?? '',
    location: resume.location ?? '',
    summary: resume.summary ?? '',
    links: resume.links,
  })

  useAutosave(form, async (value) => {
    dispatch({
      type: 'update_meta',
      patch: {
        full_name: value.full_name,
        headline: value.headline || null,
        email: value.email || null,
        phone: value.phone || null,
        location: value.location || null,
        summary: value.summary || null,
        links: value.links,
      },
    })
  })

  function setLink(index: number, patch: Partial<ResumeLink>) {
    setForm((f) => ({
      ...f,
      links: f.links.map((l, i) => (i === index ? { ...l, ...patch } : l)),
    }))
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="full_name">Full name</Label>
          <Input
            id="full_name"
            value={form.full_name}
            onChange={(e) => setForm((f) => ({ ...f, full_name: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="headline">Headline</Label>
          <Input
            id="headline"
            value={form.headline}
            onChange={(e) => setForm((f) => ({ ...f, headline: e.target.value }))}
            placeholder="Senior Software Engineer"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            value={form.email}
            onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="phone">Phone</Label>
          <Input
            id="phone"
            value={form.phone}
            onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <Label htmlFor="location">Location</Label>
          <Input
            id="location"
            value={form.location}
            onChange={(e) => setForm((f) => ({ ...f, location: e.target.value }))}
          />
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="summary">Summary</Label>
        <Textarea
          id="summary"
          rows={4}
          value={form.summary}
          onChange={(e) => setForm((f) => ({ ...f, summary: e.target.value }))}
        />
      </div>

      <div className="flex flex-col gap-2">
        <Label>Links</Label>
        {form.links.map((link, i) => (
          <div key={i} className="flex gap-2">
            <Input
              className="w-32 shrink-0"
              placeholder="Label"
              value={link.label}
              onChange={(e) => setLink(i, { label: e.target.value })}
            />
            <Input
              placeholder="https://…"
              value={link.url}
              onChange={(e) => setLink(i, { url: e.target.value })}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-9 shrink-0 text-muted-foreground hover:text-destructive"
              onClick={() => setForm((f) => ({ ...f, links: f.links.filter((_, idx) => idx !== i) }))}
            >
              <X className="size-3.5" />
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="secondary"
          size="sm"
          className="w-fit"
          onClick={() => setForm((f) => ({ ...f, links: [...f.links, { label: '', url: '' }] }))}
        >
          <Plus className="size-3.5" /> Add link
        </Button>
      </div>
    </div>
  )
}
