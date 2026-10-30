// Run from the repository root: node frontend/scripts/update-pdf-fonts.mjs
// Font binaries and their OFL licenses are self-hosted; no Google request is made by the app.
import { mkdir, writeFile } from 'node:fs/promises'
const families = ['Gelasio', 'Tinos', 'Arimo', 'Inter', 'Roboto', 'Lora', 'Merriweather', 'Open Sans']
const root = new URL('../public/fonts/pdf/', import.meta.url)
await mkdir(root, { recursive: true })
for (const family of families) {
  const slug = family.toLowerCase().replaceAll(' ', '-')
  const response = await fetch(`https://fonts.googleapis.com/css?family=${encodeURIComponent(family)}:400,700,400italic,700italic`, { headers: { 'User-Agent': 'Mozilla/5.0' } })
  if (!response.ok) throw new Error(`Font stylesheet: ${family} (${response.status})`)
  const css = await response.text()
  let downloaded = 0
  for (const match of css.matchAll(/@font-face\s*\{([^}]+)\}/g)) {
    const style = /font-style:\s*(\w+)/.exec(match[1])?.[1]
    const weight = /font-weight:\s*(\d+)/.exec(match[1])?.[1]
    const url = /src:\s*url\(([^)]+)\)/.exec(match[1])?.[1]
    if (!url || !style || !weight) throw new Error(`Incomplete font face: ${family}`)
    if (!url.endsWith('.ttf')) throw new Error(`Expected TrueType: ${url}`)
    const font = await fetch(url)
    if (!font.ok) throw new Error(`Font download: ${url}`)
    await writeFile(new URL(`${slug}-${weight}-${style}.ttf`, root), new Uint8Array(await font.arrayBuffer()))
    downloaded++
  }
  if (downloaded !== 4) throw new Error(`Expected four font faces for ${family}, got ${downloaded}`)
  let licenseText
  for (const directory of ['ofl', 'apache']) {
    for (const name of ['OFL.txt', 'LICENSE.txt']) {
      const license = await fetch(`https://raw.githubusercontent.com/google/fonts/main/${directory}/${slug.replaceAll('-', '')}/${name}`)
      if (license.ok) { licenseText = await license.text(); break }
    }
    if (licenseText) break
  }
  if (!licenseText) {
    for (const branch of ['main', 'master']) {
      const license = await fetch(`https://raw.githubusercontent.com/googlefonts/${slug.replaceAll('-', '')}/${branch}/OFL.txt`)
      if (license.ok) { licenseText = await license.text(); break }
    }
  }
  if (!licenseText) throw new Error(`Missing license: ${family}`)
  await writeFile(new URL(`${slug}-LICENSE.txt`, root), licenseText)
  console.log(`Downloaded ${family} and its license`)
}
