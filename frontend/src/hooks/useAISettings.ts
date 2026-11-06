import { useQuery } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { useFeatures } from '@/platform/features/useFeatures'
import { apiGet } from '@/lib/http'

export const AI_PROVIDERS = [
  { id: 'openai', name: 'OpenAI' },
  { id: 'anthropic', name: 'Anthropic' },
  { id: 'vercel', name: 'Vercel AI Gateway' },
  { id: 'openrouter', name: 'OpenRouter' },
] as const
export type AIProvider = typeof AI_PROVIDERS[number]['id']
export interface AIConnection {
  id: string; provider: AIProvider; model: string; key_suffix: string
  pdf_supported: boolean; validated_at: string | null
}
export interface AISettings {
  platform_available: boolean; byok_available: boolean; connections: AIConnection[]
  source: '' | 'platform' | 'byok'; provider: string; model: string
  ready: boolean; pdf_supported: boolean; reason: string
}
export interface AIModel { id: string; name: string; pdf: boolean }

export function useAISettings() {
  const { user } = useAuth()
  const { enabled, loading: flagsLoading } = useFeatures()
  const platform = enabled('platform_ai'), byok = enabled('byok_ai')
  const platformPublic = enabled('platform_ai_public')
  const query = useQuery({
    queryKey: ['ai-settings', user?.id, platform, byok, platformPublic],
    queryFn: ({ signal }) => apiGet<AISettings>('/api/ai/settings', { signal }),
    enabled: Boolean(user), staleTime: 0, gcTime: 0, retry: false,
    refetchInterval: 10_000, refetchOnWindowFocus: 'always',
  })
  const settings = !query.isError && query.isFetchedAfterMount ? query.data : undefined
  const platformAvailable = platform && settings?.platform_available === true
  const byokAvailable = byok && settings?.byok_available === true
  const selectedAvailable = settings?.source === 'platform' ? platformAvailable : settings?.source === 'byok' && byokAvailable
  const ready = selectedAvailable && settings?.ready === true
  return {
    settings, loading: flagsLoading || query.isLoading, error: query.isError,
    platformAvailable, byokAvailable, byokEnabled: byok, available: platformAvailable || byokAvailable,
    ready, pdfReady: ready && settings?.pdf_supported === true,
    label: settings?.source === 'platform' ? 'Platform AI' : AI_PROVIDERS.find(p => p.id === settings?.provider)?.name ?? 'Connect AI',
    refetch: query.refetch,
  }
}
