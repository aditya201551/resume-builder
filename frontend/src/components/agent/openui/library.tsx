/* oxlint-disable react/only-export-components -- OpenUI and the prompt CLI consume this component catalog. */
import { useId, useState } from 'react'
import { createLibrary, defineComponent, type PromptOptions } from '@openuidev/react-lang'
import { z } from 'zod/v4'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useUIInteraction } from './context'
import type { UIFormState } from '@/lib/assistantUI'

const TextBlock = defineComponent({
  name: 'TextBlock', description: 'A short plain-text explanation. No Markdown or HTML.',
  props: z.object({ text: z.string().max(1000) }),
  component: ({ props }) => <p className="text-sm leading-relaxed text-muted-foreground">{props.text}</p>,
})

const ExperienceForm = defineComponent({
  name: 'ExperienceForm', description: 'Collect one NEW work experience for user review. Prefill only facts the user already supplied; use empty strings for unknown facts. Dates are YYYY-MM or empty.',
  props: z.object({
    company: z.string().max(200), role: z.string().max(200), location: z.string().max(200),
    startDate: z.string().max(7), endDate: z.string().max(7), current: z.boolean(), details: z.string().max(12000),
  }),
  component: function ExperienceFields({ props }) {
    const context = useUIInteraction()
    const id = useId()
    const [values, setValues] = useState<UIFormState>(() => Object.keys(context.formState).length ? context.formState : {
      company: props.company, title: props.role, location: props.location, startDate: props.startDate, endDate: props.endDate, current: props.current, content: props.details,
    })
    if (context.ui.kind !== 'experience_form') throw new Error('Experience form is not available in this card.')
    function change(key: string, value: string | boolean) {
      const next = { ...values, [key]: value }
      setValues(next)
      context.onStateChange(next)
    }
    return <form className="flex flex-col gap-3" onSubmit={(event) => { event.preventDefault(); context.onReview({ type: 'experience', values }) }}>
      <fieldset disabled={context.disabled} className="grid min-w-0 grid-cols-1 gap-3 sm:grid-cols-2">
        {([['company', 'Company', 'text'], ['title', 'Role', 'text'], ['location', 'Location', 'text'], ['startDate', 'Start date', 'month'], ['endDate', 'End date', 'month']] as const).map(([key, label, type]) => <div key={key} className="min-w-0">
          <label htmlFor={`${id}-${key}`} className="mb-1 block text-xs font-medium">{label}{['company', 'title'].includes(key) ? ' *' : ''}</label>
          <Input id={`${id}-${key}`} type={type} value={String(values[key] ?? '')} maxLength={type === 'text' ? 200 : undefined}
            required={key === 'company' || key === 'title'} disabled={context.disabled || (key === 'endDate' && values.current === true)}
            onChange={(event) => change(key, event.target.value)} />
        </div>)}
        <label className="flex items-center gap-2 self-end py-2 text-xs"><input type="checkbox" checked={values.current === true} onChange={(event) => change('current', event.target.checked)} /> I currently work here</label>
        <div className="sm:col-span-2">
          <label htmlFor={`${id}-content`} className="mb-1 block text-xs font-medium">Responsibilities and achievements</label>
          <Textarea id={`${id}-content`} value={String(values.content ?? '')} maxLength={12000} rows={4} placeholder="What did you do, and what changed because of your work?" onChange={(event) => change('content', event.target.value)} />
        </div>
      </fieldset>
      <p className="text-[11px] leading-relaxed text-muted-foreground">Leave dates blank if you're unsure. You'll review the proposed entry before it is added.</p>
      <Button type="submit" size="sm" className="w-fit" disabled={context.disabled}>Review experience</Button>
    </form>
  },
})

const RewriteComparison = defineComponent({
  name: 'RewriteComparison', description: 'Offer 2 or 3 grounded alternatives for the one rewrite target bound by show_resume_ui. The original text comes from the app. No invented metrics, skills or facts.',
  props: z.object({ options: z.array(z.object({ label: z.string().min(1).max(80), text: z.string().min(1).max(16000) })).min(2).max(3) }),
  component: function RewriteChoices({ props }) {
    const context = useUIInteraction()
    const id = useId()
    const [edits, setEdits] = useState<UIFormState>(context.formState)
    if (context.ui.kind !== 'rewrite_comparison' || !context.ui.target) throw new Error('Rewrite target is missing.')
    return <div className="flex flex-col gap-3">
      <div className="rounded-xl bg-background/70 p-3"><p className="mb-1 text-xs font-medium">Current text</p><p className="whitespace-pre-wrap break-words text-xs leading-relaxed text-muted-foreground">{context.ui.target.original || 'No text yet.'}</p></div>
      {props.options.map((option, index) => <div key={index} className="flex min-w-0 flex-col gap-2 rounded-xl border border-border bg-card p-3">
        <label htmlFor={`${id}-${index}`} className="text-xs font-medium">{option.label}</label>
        <Textarea id={`${id}-${index}`} value={String(edits[`option${index}`] ?? option.text)} maxLength={16000} rows={4} disabled={context.disabled} onChange={(event) => {
          const next = { ...edits, [`option${index}`]: event.target.value }; setEdits(next); context.onStateChange(next)
        }} />
        <Button type="button" variant="outline" size="sm" className="w-fit" disabled={context.disabled} onClick={() => context.onReview({ type: 'rewrite', text: String(edits[`option${index}`] ?? option.text) })}>Review this version</Button>
      </div>)}
    </div>
  },
})

const AssistantPanel = defineComponent({
  name: 'AssistantPanel', description: 'Root resume assistant card. Include one interactive form OR one rewrite comparison, optionally preceded by a TextBlock.',
  props: z.object({ title: z.string().min(1).max(120), children: z.array(z.union([TextBlock.ref, ExperienceForm.ref, RewriteComparison.ref])).min(1).max(3) }),
  component: ({ props, renderNode }) => <section className="flex min-w-0 flex-col gap-4 rounded-2xl border border-primary/15 bg-secondary/30 p-4"><h3 className="text-sm font-semibold">{props.title}</h3>{renderNode(props.children)}</section>,
})

export const assistantLibrary = createLibrary({ root: 'AssistantPanel', components: [AssistantPanel, TextBlock, ExperienceForm, RewriteComparison] })
export const assistantPromptOptions: PromptOptions = {
  toolCalls: false, bindings: false, builtinFunctions: false,
  additionalRules: [
    'Return the program only as the code argument of show_resume_ui, never in normal chat text.',
    'Use one AssistantPanel root and exactly one ExperienceForm or one RewriteComparison. No Query, Mutation, actions, functions or external URLs.',
    'The experience_form kind requires ExperienceForm; rewrite_comparison requires RewriteComparison. Do not mix the kinds.',
  ],
}
