import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Check, KeyRound, Loader2, Trash2 } from 'lucide-react'
import { AI_PROVIDERS, useAISettings, type AIModel, type AIProvider } from '@/hooks/useAISettings'
import { apiDelete, apiGet, apiPost, apiPut, HttpError } from '@/lib/http'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { useFeatures } from '@/platform/features/useFeatures'

function publicError(error: unknown) {
  if (error instanceof HttpError) {
    const start = error.message.indexOf('{')
    if (start >= 0) { try { return (JSON.parse(error.message.slice(start)) as { message: string }).message } catch { /* use safe fallback */ } }
  }
  return 'That change could not be completed. Please try again.'
}

export function AIConnections({ onConnected }: { onConnected?: () => void } = {}) {
  const ai = useAISettings()
  const queryClient = useQueryClient()
  const [keys, setKeys] = useState<Partial<Record<AIProvider, string>>>({})
  const [models, setModels] = useState<Partial<Record<AIProvider, AIModel[]>>>({})
  const [selection, setSelection] = useState<Partial<Record<AIProvider, string>>>({})
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  async function run(id: string, action: () => Promise<unknown>, message = '') {
    setBusy(id); setError(''); setNotice('')
    try {
      await action()
      await queryClient.invalidateQueries({ queryKey: ['ai-settings'] })
      setNotice(message)
    } catch (error) { setError(publicError(error)) }
    finally { setBusy(null) }
  }
  const settings = ai.settings
  return <div className="flex flex-col gap-6">
    <p className="text-sm text-muted-foreground">Choose who supplies the AI connection for your assistant and PDF imports. Your keys are encrypted on the server and never included in resume drafts.</p>
    {ai.loading && <p role="status" className="text-sm">Checking AI availability…</p>}
    {ai.error && <div role="alert"><p>AI settings could not be loaded.</p><Button variant="outline" onClick={() => void ai.refetch()}>Try again</Button></div>}
    {error && <p role="alert" className="rounded-xl bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}
    {notice && <p role="status" className="text-sm">{notice}</p>}
    {!ai.loading && !ai.error && !ai.available && <p className="rounded-xl bg-secondary p-4 text-sm">AI is currently unavailable for this account. Your saved connections are retained; you can delete them below.</p>}
    {settings?.source && <p className="text-sm"><strong>Selected:</strong> {ai.label}{settings.source === 'byok' && settings.model ? ` · ${settings.model}` : ''}{!ai.ready ? ' · unavailable or setup needed' : settings.pdf_supported ? ' · chat and PDF import' : ' · chat only'}</p>}
    {ai.platformAvailable && <section className="flex flex-wrap items-center justify-between gap-3 rounded-xl border p-4">
      <div><h2 className="font-semibold">Platform AI</h2><p className="text-sm text-muted-foreground">Provided by Resume Builder.</p></div>
      <Button variant={settings?.source === 'platform' ? 'secondary' : 'outline'} disabled={Boolean(busy)} onClick={() => void run('platform', async () => { await apiPut('/api/ai/preferences', { source: 'platform' }); onConnected?.() }, 'Platform AI selected.')}>
        {settings?.source === 'platform' && <Check className="size-4" />} Use Platform AI
      </Button>
    </section>}
    {AI_PROVIDERS.map(provider => {
      const connection = settings?.connections.find(c => c.provider === provider.id)
      if (!ai.byokAvailable && !connection) return null
      const chosen = selection[provider.id] ?? connection?.model ?? ''
      const changedModel = chosen !== (connection?.model ?? '')
      const current = settings?.source === 'byok' && settings.provider === provider.id
      const url = `/api/ai/connections/${provider.id}`
      return <section key={provider.id} className="flex flex-col gap-3 rounded-xl border p-4">
        <div className="flex items-start justify-between gap-3">
          <div><h2 className="font-semibold">{provider.name}</h2><p className="text-xs text-muted-foreground">{connection ? `Key ending ${connection.key_suffix}${connection.validated_at ? ' · connection tested' : ' · test needed'}` : 'Connect your own account'}</p></div>
          {connection && <Button size="icon" variant="ghost" disabled={Boolean(busy)} aria-label={`Delete ${provider.name} key`} onClick={() => void run(provider.id, async () => { await apiDelete(url); setModels(m => ({ ...m, [provider.id]: [] })); setSelection(m => ({ ...m, [provider.id]: '' })); setKeys(k => ({ ...k, [provider.id]: '' })) }, 'Key deleted.')}><Trash2 className="size-4" /></Button>}
        </div>
        {ai.byokAvailable && <>
          <form className="flex flex-col gap-2 sm:flex-row" onSubmit={event => {
            event.preventDefault()
            const key = keys[provider.id] ?? ''
            void run(provider.id, async () => {
              await apiPut(url, { api_key: key, model: '' })
              setKeys(k => ({ ...k, [provider.id]: '' }))
              setSelection(m => ({ ...m, [provider.id]: '' }))
              setModels(m => ({ ...m, [provider.id]: [] }))
            }, 'Encrypted key saved. Load models to continue.')
          }}>
            <Label className="sr-only" htmlFor={`key-${provider.id}`}>{provider.name} API key</Label>
            <Input id={`key-${provider.id}`} type="password" autoComplete="off" spellCheck={false} value={keys[provider.id] ?? ''} onChange={event => setKeys(k => ({ ...k, [provider.id]: event.target.value }))} placeholder={provider.id === 'vercel' ? 'Vercel AI Gateway API key' : connection ? 'Enter a replacement API key' : `${provider.name} API key`} disabled={Boolean(busy)} />
            <Button type="submit" variant="outline" disabled={Boolean(busy) || !(keys[provider.id] ?? '').trim()}>{connection ? 'Replace key' : 'Save key'}</Button>
          </form>
          {connection && <>
            <Button variant="outline" className="w-fit" disabled={Boolean(busy)} onClick={() => void run(provider.id, async () => { const list = await apiGet<AIModel[]>(`${url}/models`); setModels(m => ({ ...m, [provider.id]: list })); if (list.length === 0) setNotice('No compatible tool-calling models were found for this account.') })}>Load available models</Button>
            {(models[provider.id]?.length ?? 0) > 0 && <div className="flex flex-col gap-2">
              <Label htmlFor={`model-${provider.id}`}>Model</Label>
              <select id={`model-${provider.id}`} className="h-10 w-full rounded-md border bg-background px-3 text-sm" value={chosen} disabled={Boolean(busy)} onChange={event => setSelection(m => ({ ...m, [provider.id]: event.target.value }))}>
                <option value="">Choose a model</option>
                {models[provider.id]!.map(model => <option key={model.id} value={model.id}>{model.name} · {model.pdf ? 'chat + PDF' : 'chat'}</option>)}
              </select>
            </div>}
            {!models[provider.id]?.length && connection.model && <p className="break-all text-sm">{connection.model} · {connection.pdf_supported ? 'chat and native PDF input' : 'chat only'}</p>}
            <div className="flex flex-wrap gap-2">
              {changedModel && <Button variant="outline" disabled={Boolean(busy) || !chosen} onClick={() => void run(provider.id, () => apiPut(url, { model: chosen }), 'Model saved. Test the connection next.')}>Save model</Button>}
              <Button variant="outline" disabled={Boolean(busy) || !connection.model || changedModel} onClick={() => void run(provider.id, () => apiPost(`${url}/test`), 'Connection test passed. You can select this connection.')}>
                {busy === provider.id && <Loader2 className="size-4 animate-spin" />} Test connection
              </Button>
              <Button disabled={Boolean(busy) || !connection.validated_at || changedModel} variant={current ? 'secondary' : 'default'} onClick={() => void run(provider.id, async () => { await apiPut('/api/ai/preferences', { source: 'byok', provider: provider.id }); onConnected?.() }, `${provider.name} selected.`)}>{current && <Check className="size-4" />} Use this connection</Button>
            </div>
            <p className="text-xs text-muted-foreground">Testing sends a small synthetic tool-call request and may incur a provider charge. Model PDF capability is advertised by the provider; extraction quality can vary.</p>
          </>}
        </>}
      </section>
    })}
  </div>
}

export function AIConnectionDialog({ open, onOpenChange, onConnected }: { open: boolean; onOpenChange: (open: boolean) => void; onConnected?: () => void }) {
  const { enabled } = useFeatures()
  if (!enabled('byok_ai')) return null
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-2xl"><DialogHeader><DialogTitle>AI connections</DialogTitle><DialogDescription>Select Platform AI or connect your own provider.</DialogDescription></DialogHeader>{open && <AIConnections onConnected={onConnected} />}</DialogContent></Dialog>
}

export function AIConnectionSelector({ onConnected }: { onConnected?: () => void } = {}) {
  const ai = useAISettings()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [selecting, setSelecting] = useState(false)
  const [error, setError] = useState('')
  if (!ai.byokEnabled) {
    if (!ai.platformAvailable || ai.ready) return null
    return <div className="flex flex-col gap-2">
      <Button size="sm" variant="outline" disabled={selecting} onClick={async () => {
        setSelecting(true); setError('')
        try {
          await apiPut('/api/ai/preferences', { source: 'platform' })
          await queryClient.invalidateQueries({ queryKey: ['ai-settings'] })
          onConnected?.()
        } catch (error) { setError(publicError(error)) }
        finally { setSelecting(false) }
      }}>{selecting && <Loader2 className="size-3.5 animate-spin" />}Use Platform AI</Button>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    </div>
  }
  const label = ai.ready ? `${ai.label}${ai.settings?.source === 'byok' && ai.settings.model ? ` · ${ai.settings.model}` : ''}` : 'Configure AI connection'
  return <><Button variant="ghost" size="sm" className="max-w-full justify-start" onClick={() => setOpen(true)}><KeyRound className="size-3.5" /><span className="truncate">{label}</span></Button><AIConnectionDialog open={open} onOpenChange={setOpen} onConnected={onConnected} /></>
}

export default function AISettingsPage() {
  return <div className="mx-auto w-full max-w-3xl px-5 py-8"><h1 className="mb-6 text-2xl font-semibold">AI connections</h1><AIConnections /></div>
}
