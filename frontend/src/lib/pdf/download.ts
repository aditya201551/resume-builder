export function downloadPdfBlob(blob: Blob, label: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  const filename = [...label].filter((char) => char.charCodeAt(0) >= 32 && !'<>:"/\\|?*'.includes(char)).join('').trim()
  anchor.download = `${filename || 'resume'}.pdf`
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  setTimeout(() => URL.revokeObjectURL(url), 60000)
}
