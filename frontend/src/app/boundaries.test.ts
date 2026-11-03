import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = fileURLToPath(new URL('..', import.meta.url))
const jobs = path.join(source, 'features/jobs')

function files(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const file = path.join(directory, entry.name)
    return entry.isDirectory() ? files(file) : /\.[jt]sx?$/.test(file) ? [file] : []
  })
}

describe('Jobs module boundary', () => {
  it('allows application imports only through the public module entry point', () => {
    const violations: string[] = []
    for (const file of files(source)) {
      if (file.startsWith(jobs + path.sep) || file.endsWith('.test.ts') || file.endsWith('.test.tsx')) continue
      for (const match of readFileSync(file, 'utf8').matchAll(/(?:from\s*|import\s*\(\s*|import\s*)['"]([^'"]+)['"]/g)) {
        const specifier = match[1]
        const resolved = specifier.startsWith('@/') ? path.join(source, specifier.slice(2))
          : specifier.startsWith('.') ? path.resolve(path.dirname(file), specifier) : ''
        if (resolved.startsWith(jobs + path.sep) && ![path.join(jobs, 'index'), path.join(jobs, 'index.ts')].includes(resolved)) {
          violations.push(`${path.relative(source, file)} imports ${specifier}`)
        }
      }
    }
    expect(violations).toEqual([])
  })

  it('keeps Jobs independent of resume pages and editor components', () => {
    const violations = files(jobs).filter(file => !file.includes('.test.')).flatMap(file => {
      const imports = [...readFileSync(file, 'utf8').matchAll(/from\s*['"](@\/(?:pages|components\/(?:editor|agent))\/[^'"]+)['"]/g)]
      return imports.map(match => `${path.relative(source, file)} imports ${match[1]}`)
    })
    expect(violations).toEqual([])
  })
})
