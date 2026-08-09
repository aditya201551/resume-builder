export class HttpError extends Error {
  status: number
  constructor(method: string, url: string, status: number, body: string) {
    super(`${method} ${url} failed: ${status} ${body}`)
    this.name = 'HttpError'
    this.status = status
  }
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new HttpError(method, url, res.status, text)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const apiGet = <T>(url: string) => request<T>('GET', url)
export const apiPost = <T>(url: string, body?: unknown) => request<T>('POST', url, body)
export const apiPatch = <T>(url: string, body?: unknown) => request<T>('PATCH', url, body)
export const apiPut = <T>(url: string, body?: unknown) => request<T>('PUT', url, body)
export const apiDelete = (url: string) => request<void>('DELETE', url)

export async function apiDownload(url: string, fallbackFilename: string): Promise<void> {
  const res = await fetch(url, { credentials: 'include' })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(`GET ${url} failed: ${res.status} ${text}`)
  }
  const disposition = res.headers.get('Content-Disposition')
  const filename = disposition?.match(/filename="(.+)"/)?.[1] ?? fallbackFilename
  const blob = await res.blob()
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(objectUrl)
}
