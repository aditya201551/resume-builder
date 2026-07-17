import type { User } from '../types/user'

export async function getMe(): Promise<User | null> {
  const res = await fetch('/api/auth/me', { credentials: 'include' })
  if (res.status === 401) return null
  if (!res.ok) throw new Error(`GET /api/auth/me failed: ${res.status}`)
  return res.json()
}

export async function logout(): Promise<void> {
  const res = await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' })
  if (!res.ok) throw new Error(`POST /api/auth/logout failed: ${res.status}`)
}

export function loginUrl(provider: 'google' | 'github'): string {
  return `/api/auth/${provider}/login`
}
