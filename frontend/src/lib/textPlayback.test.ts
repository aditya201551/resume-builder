import { afterEach, expect, it, vi } from 'vitest'
import { CHAT_WORD_INTERVAL_MS, createTextPlayback } from './textPlayback'

afterEach(() => vi.useRealTimers())

it('paces a fast burst and drains after network completion without dumping the tail', async () => {
  vi.useFakeTimers()
  const words: string[] = []
  const playback = createTextPlayback((word) => words.push(word), () => false)
  playback.write('One two three four')
  let done = false
  const finished = playback.finish().then(() => { done = true })
  expect(words).toEqual([])
  await vi.advanceTimersByTimeAsync(CHAT_WORD_INTERVAL_MS * 2)
  expect(words).toEqual(['One ', 'two '])
  expect(done).toBe(false)
  await vi.runAllTimersAsync()
  await finished
  expect(words.join('')).toBe('One two three four')
})

it('preserves split words, Markdown, whitespace and ordering across tool boundaries', async () => {
  vi.useFakeTimers()
  const output: string[] = []
  const playback = createTextPlayback((word) => output.push(word), () => false)
  playback.write('**stren')
  await vi.advanceTimersByTimeAsync(500)
  expect(output).toEqual([])
  playback.write('gthen**\n\n- this')
  playback.boundary(() => output.push('[tool]'))
  playback.write(' Next.')
  const finished = playback.finish()
  await vi.runAllTimersAsync()
  await finished
  expect(output).toEqual(['**strengthen**\n\n', '- ', 'this', '[tool]', ' Next.'])
})

it('cancels buffered words and actions without leaking them into the next response', async () => {
  vi.useFakeTimers()
  const output: string[] = []
  const action = vi.fn()
  const playback = createTextPlayback((word) => output.push(word), () => false)
  playback.write('One two three')
  playback.boundary(action)
  const finished = playback.finish()
  await vi.advanceTimersByTimeAsync(CHAT_WORD_INTERVAL_MS)
  playback.cancel()
  await finished
  await vi.runAllTimersAsync()
  expect(output).toEqual(['One '])
  expect(action).not.toHaveBeenCalled()
  expect(vi.getTimerCount()).toBe(0)
})

it('shows text immediately with reduced motion and drains if that preference changes', async () => {
  vi.useFakeTimers()
  const output: string[] = []
  let immediate = false
  const playback = createTextPlayback((word) => output.push(word), () => immediate)
  playback.write('One two ')
  immediate = true
  playback.write('partial')
  await vi.advanceTimersByTimeAsync(CHAT_WORD_INTERVAL_MS)
  await playback.finish()
  expect(output.join('')).toBe('One two partial')
  expect(vi.getTimerCount()).toBe(0)
  const direct = createTextPlayback((word) => output.push(word), () => true)
  direct.write(' now')
  expect(output.join('')).toBe('One two partial now')
})
