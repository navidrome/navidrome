import { describe, expect, it, vi } from 'vitest'
import {
  buildKaraokeLines,
  getPreferredLyricLanguage,
  hasStructuredLyricContent,
  selectLyricLayers,
  utf8ByteRangeToCodeUnitRange,
} from './lyrics'

const timed = (value, start, end, extra = {}) => ({
  ...extra,
  start,
  ...(end == null ? {} : { end }),
  value,
})

const structured = (kind, lang, value) => ({
  kind,
  lang,
  synced: true,
  line: [timed(value, 0)],
})

const cueLine = (index, value, start, end, cue, extra = {}) => ({
  index,
  ...timed(value, start, end, extra),
  cue,
})

describe('lyrics helpers', () => {
  it('selects main, pronunciation, and translation layers by kind and language', () => {
    const layers = selectLyricLayers(
      [
        structured('translation', 'es', 'Hola'),
        structured('main', 'en-US', 'Hello'),
        structured('pronunciation', 'en', 'heh-low'),
      ],
      'en',
    )

    expect(layers.main.value || layers.main.line[0].value).toBe('Hello')
    expect(layers.pronunciation.line[0].value).toBe('heh-low')
    expect(layers.translation.line[0].value).toBe('Hola')
  })

  it('prefers cue-timed lyrics when line starts are omitted', () => {
    const layers = selectLyricLayers(
      [
        {
          kind: 'main',
          lang: 'en',
          synced: false,
          line: [{ value: 'Plain lyrics' }],
        },
        {
          kind: 'main',
          lang: 'en',
          synced: true,
          line: [{ value: 'Cue-timed lyrics' }],
          cueLine: [
            {
              index: 0,
              value: 'Cue-timed lyrics',
              cue: [{ start: 1000, value: 'Cue-timed lyrics' }],
            },
          ],
        },
      ],
      'en',
    )

    expect(layers.main.line[0].value).toBe('Cue-timed lyrics')
  })

  it('prefers the requested language before timed variants', () => {
    const layers = selectLyricLayers(
      [
        {
          kind: 'main',
          lang: 'en',
          synced: false,
          line: [{ value: 'Preferred plain lyrics' }],
        },
        structured('main', 'ja', '別の言語'),
      ],
      'en-US',
    )

    expect(layers.main.line[0].value).toBe('Preferred plain lyrics')
  })

  it('matches language tags with multiple underscore separators', () => {
    const layers = selectLyricLayers(
      [
        structured('main', 'zh-Hans', '简体'),
        structured('main', 'zh-Hant-TW', '繁體'),
      ],
      'zh_Hant_TW',
    )

    expect(layers.main.line[0].value).toBe('繁體')
  })

  it('matches ISO 639-2 lyric languages to regional browser locales', () => {
    const lyrics = [
      structured('main', 'spa', 'Español'),
      structured('main', 'eng', 'English'),
      structured('main', 'por', 'Português'),
    ]

    expect(selectLyricLayers(lyrics, 'en-US').main.line[0].value).toBe(
      'English',
    )
    expect(selectLyricLayers(lyrics, 'pt-BR').main.line[0].value).toBe(
      'Português',
    )
  })

  it('resolves UTF-8 byte ranges without confusing repeated words', () => {
    const text = 'caf\u00e9 caf\u00e9'

    expect(utf8ByteRangeToCodeUnitRange(text, 0, 4)).toMatchObject({
      text: 'caf\u00e9',
    })
    expect(utf8ByteRangeToCodeUnitRange(text, 6, 10)).toMatchObject({
      text: 'caf\u00e9',
      start: 5,
    })
  })

  it('applies structured lyric offsets to line and cue timing', () => {
    const delayed = buildKaraokeLines({
      synced: true,
      offset: 500,
      line: [timed('Delayed line', 1000, 3000)],
      cueLine: [
        cueLine(0, 'Delayed line', 1000, 3000, [
          timed('Delayed ', 1000, 2000, { byteStart: 0, byteEnd: 7 }),
          timed('line', 2000, 3000, { byteStart: 8, byteEnd: 11 }),
        ]),
      ],
    })
    const advanced = buildKaraokeLines({
      synced: true,
      offset: -250,
      line: [timed('Advanced line', 1000)],
    })

    expect(delayed[0].start).toBe(1500)
    expect(delayed[0].end).toBe(3500)
    expect(delayed[0].tokens[0].start).toBe(1500)
    expect(delayed[0].tokens[0].end).toBe(2500)
    expect(advanced[0].start).toBe(750)
  })

  it('keeps base lyric lines that do not have word-level cue lines', () => {
    const lines = buildKaraokeLines({
      synced: true,
      line: [
        timed('Word timed', 1000),
        timed('Plain timed line', 2000),
        timed('More words', 3000),
      ],
      cueLine: [
        cueLine(0, 'Word timed', 1000, null, [
          timed('Word ', 1000, null, { byteStart: 0, byteEnd: 4 }),
          timed('timed', 1500, null, { byteStart: 5, byteEnd: 9 }),
        ]),
        cueLine(2, 'More words', 3000, null, [
          timed('More ', 3000, null, { byteStart: 0, byteEnd: 4 }),
          timed('words', 3500, null, { byteStart: 5, byteEnd: 9 }),
        ]),
      ],
    })

    expect(lines.map((line) => line.value)).toEqual([
      'Word timed',
      'Plain timed line',
      'More words',
    ])
    expect(lines[1].tokens).toEqual([])
  })

  it('preserves repeated ELRC-style cue timing and trailing cue ends', () => {
    const repeatedLine = (index, start) =>
      cueLine(index, 'Hello world', start, start + 900, [
        timed('Hello ', start + 100, start + 500, {
          byteStart: 0,
          byteEnd: 5,
        }),
        timed('world', start + 500, start + 900, {
          byteStart: 6,
          byteEnd: 10,
        }),
      ])
    const lines = buildKaraokeLines({
      synced: true,
      line: [
        timed('Hello world', 10000, 10900),
        timed('Hello world', 30000, 30900),
      ],
      cueLine: [repeatedLine(0, 10000), repeatedLine(1, 30000)],
    })

    expect(lines).toHaveLength(2)
    expect(lines[0].tokens[1].end).toBe(10900)
    expect(lines[1].tokens[0].start).toBe(30100)
  })

  it('keeps multiline SRT, TTML, and plain text values intact', () => {
    const lines = buildKaraokeLines({
      synced: true,
      line: [{ start: 1000, value: 'first line\nsecond line' }],
    })

    expect(lines[0].value).toBe('first line\nsecond line')
  })

  it('keeps adjacent TTML syllable cue tokens in order', () => {
    const lines = buildKaraokeLines({
      synced: true,
      line: [timed('konni', 2000)],
      cueLine: [
        cueLine(0, 'konni', 2000, 2600, [
          timed('ko', 2000, 2300, { byteStart: 0, byteEnd: 1 }),
          timed('nni', 2300, 2600, { byteStart: 2, byteEnd: 4 }),
        ]),
      ],
    })

    expect(lines[0].tokens.map((token) => token.value)).toEqual(['ko', 'nni'])
  })

  it('keeps same-index agent cue lines as ordered voice lanes', () => {
    const agentCue = (value, agentId, start, end) =>
      cueLine(
        0,
        value,
        start,
        end,
        [timed(value, start, end, { byteStart: 0, byteEnd: value.length - 1 })],
        { agentId },
      )
    const lines = buildKaraokeLines({
      synced: true,
      agents: [
        { id: 'lead', role: 'main' },
        { id: 'all', role: 'group' },
        { id: 'lead-bg', role: 'bg' },
      ],
      line: [timed('Lead all echo', 1000, 4000)],
      cueLine: [
        agentCue('echo', 'lead-bg', 2000, 3000),
        agentCue('all', 'all', 1500, 2500),
        agentCue('Lead', 'lead', 1000, 2000),
      ],
    })

    expect(lines).toHaveLength(1)
    expect(lines[0].lanes).toHaveLength(3)
    expect(lines[0].lanes.map((lane) => lane.agentRole)).toEqual([
      'main',
      'group',
      'bg',
    ])
    expect(lines[0].tokens.map((token) => token.value)).toEqual([
      'Lead',
      'all',
      'echo',
    ])
  })

  it('uses whitespace-aware text when cue-only fallback values are needed', () => {
    const lines = buildKaraokeLines({
      synced: true,
      cueLine: [
        cueLine(0, '', 1000, 3000, [
          timed('Lead', 1000, 2000),
          timed('all', 2000, 3000),
          timed('echo', 3000, 3500),
        ]),
      ],
    })

    expect(lines[0].value).toBe('Lead all echo')
  })

  it('preserves untimed lyric rows between surrounding timed rows', () => {
    const lines = buildKaraokeLines({
      synced: true,
      line: [
        { start: 1000, value: 'First verse' },
        { value: '[instrumental]' },
        { start: 2000, value: 'Second verse' },
      ],
    })

    expect(lines.map((line) => line.value)).toEqual([
      'First verse',
      '[instrumental]',
      'Second verse',
    ])
  })

  it('preserves source display order independently from timestamps', () => {
    const lines = buildKaraokeLines({
      synced: true,
      line: [
        { start: 3000, value: 'Displayed first' },
        { start: 1000, value: 'Displayed second' },
        { start: 2000, value: 'Displayed third' },
      ],
    })

    expect(lines.map((line) => line.value)).toEqual([
      'Displayed first',
      'Displayed second',
      'Displayed third',
    ])
    expect(lines.map((line) => line.start)).toEqual([3000, 1000, 2000])
  })

  it('preserves timed blank rows as non-renderable timing markers', () => {
    const lines = buildKaraokeLines({
      synced: true,
      line: [
        { start: 1000, value: 'Before pause' },
        { start: 2000, value: '' },
        { start: 4000, value: 'After pause' },
      ],
    })

    expect(lines).toHaveLength(3)
    expect(lines[1]).toMatchObject({
      start: 2000,
      value: '',
      renderable: false,
    })
  })

  it('falls back to the browser language when locale storage is unavailable', () => {
    const storage = vi
      .spyOn(Storage.prototype, 'getItem')
      .mockImplementation(() => {
        throw new DOMException('Access denied', 'SecurityError')
      })

    expect(getPreferredLyricLanguage()).toBe(navigator.language)
    storage.mockRestore()
  })

  it('treats instrumental empty lyrics as no renderable content', () => {
    expect(
      hasStructuredLyricContent({
        kind: 'main',
        lang: 'en',
        synced: false,
        line: [],
      }),
    ).toBe(false)
  })
})
