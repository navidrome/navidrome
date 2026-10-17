import { describe, expect, it } from 'vitest'
import { formatSourceValue, getSourceName } from './lyricsSource'

describe('lyrics source labels', () => {
  it('formats known formats and slug-like values for display', () => {
    expect(formatSourceValue('ttml')).toBe('TTML')
    expect(formatSourceValue('elrc')).toBe('ELRC')
    expect(formatSourceValue('plain')).toBe('Plain text')
    expect(formatSourceValue('apple-lyrically')).toBe('Apple Lyrically')
    expect(formatSourceValue('  ')).toBe('')
  })

  it('names built-in, plugin, and unknown sources', () => {
    expect(getSourceName({ type: 'embedded' }, {})).toBe('Embedded lyrics')
    expect(getSourceName({ type: 'sidecar', format: 'lrc' }, {})).toBe(
      'Sidecar file',
    )
    expect(
      getSourceName(
        { type: 'plugin', name: 'Better Lyrics', provider: 'NetEase' },
        {},
      ),
    ).toBe('Better Lyrics')
    expect(getSourceName({ type: 'plugin' }, {})).toBe('Lyrics plugin')
    expect(getSourceName(null, {})).toBe('Lyrics')
  })
})
