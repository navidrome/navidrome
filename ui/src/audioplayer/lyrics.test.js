import { describe, expect, it } from 'vitest'
import {
  buildKaraokeLines,
  hasStructuredLyricContent,
  resolveKaraokeTokenWindow,
  selectLyricLayers,
  utf8ByteRangeToCodeUnitRange,
} from './lyrics'
import {
  resolveLyricsSidebarState,
  toggleLayerPreference,
} from './lyricsSidebarState'

describe('lyrics helpers', () => {
  it('selects main, pronunciation, and translation layers by kind and language', () => {
    const layers = selectLyricLayers(
      [
        {
          kind: 'translation',
          lang: 'es',
          synced: true,
          line: [{ start: 0, value: 'Hola' }],
        },
        {
          kind: 'main',
          lang: 'en-US',
          synced: true,
          line: [{ start: 0, value: 'Hello' }],
        },
        {
          kind: 'pronunciation',
          lang: 'en',
          synced: true,
          line: [{ start: 0, value: 'heh-low' }],
        },
      ],
      'en',
    )

    expect(layers.main.value || layers.main.line[0].value).toBe('Hello')
    expect(layers.pronunciation.line[0].value).toBe('heh-low')
    expect(layers.translation.line[0].value).toBe('Hola')
  })

  it('matches language tags with multiple underscore separators', () => {
    const layers = selectLyricLayers(
      [
        {
          kind: 'main',
          lang: 'zh-Hans',
          synced: true,
          line: [{ start: 0, value: '简体' }],
        },
        {
          kind: 'main',
          lang: 'zh-Hant-TW',
          synced: true,
          line: [{ start: 0, value: '繁體' }],
        },
      ],
      'zh_Hant_TW',
    )

    expect(layers.main.line[0].value).toBe('繁體')
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
      line: [{ start: 1000, end: 3000, value: 'Delayed line' }],
      cueLine: [
        {
          index: 0,
          start: 1000,
          end: 3000,
          value: 'Delayed line',
          cue: [
            {
              start: 1000,
              end: 2000,
              value: 'Delayed ',
              byteStart: 0,
              byteEnd: 7,
            },
            {
              start: 2000,
              end: 3000,
              value: 'line',
              byteStart: 8,
              byteEnd: 11,
            },
          ],
        },
      ],
    })
    const advanced = buildKaraokeLines({
      synced: true,
      offset: -250,
      line: [{ start: 1000, value: 'Advanced line' }],
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
        { start: 1000, value: 'Word timed' },
        { start: 2000, value: 'Plain timed line' },
        { start: 3000, value: 'More words' },
      ],
      cueLine: [
        {
          index: 0,
          start: 1000,
          value: 'Word timed',
          cue: [
            { start: 1000, value: 'Word ', byteStart: 0, byteEnd: 4 },
            { start: 1500, value: 'timed', byteStart: 5, byteEnd: 9 },
          ],
        },
        {
          index: 2,
          start: 3000,
          value: 'More words',
          cue: [
            { start: 3000, value: 'More ', byteStart: 0, byteEnd: 4 },
            { start: 3500, value: 'words', byteStart: 5, byteEnd: 9 },
          ],
        },
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
    const lines = buildKaraokeLines({
      synced: true,
      line: [
        { start: 10000, end: 10900, value: 'Hello world' },
        { start: 30000, end: 30900, value: 'Hello world' },
      ],
      cueLine: [
        {
          index: 0,
          start: 10000,
          end: 10900,
          value: 'Hello world',
          cue: [
            {
              start: 10100,
              end: 10500,
              value: 'Hello ',
              byteStart: 0,
              byteEnd: 5,
            },
            {
              start: 10500,
              end: 10900,
              value: 'world',
              byteStart: 6,
              byteEnd: 10,
            },
          ],
        },
        {
          index: 1,
          start: 30000,
          end: 30900,
          value: 'Hello world',
          cue: [
            {
              start: 30100,
              end: 30500,
              value: 'Hello ',
              byteStart: 0,
              byteEnd: 5,
            },
            {
              start: 30500,
              end: 30900,
              value: 'world',
              byteStart: 6,
              byteEnd: 10,
            },
          ],
        },
      ],
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
      line: [{ start: 2000, value: 'konni' }],
      cueLine: [
        {
          index: 0,
          start: 2000,
          end: 2600,
          value: 'konni',
          cue: [
            { start: 2000, end: 2300, value: 'ko', byteStart: 0, byteEnd: 1 },
            {
              start: 2300,
              end: 2600,
              value: 'nni',
              byteStart: 2,
              byteEnd: 4,
            },
          ],
        },
      ],
    })

    expect(lines[0].tokens.map((token) => token.value)).toEqual(['ko', 'nni'])
  })

  it('keeps same-index agent cue lines as ordered voice lanes', () => {
    const lines = buildKaraokeLines({
      synced: true,
      agents: [
        { id: 'lead', role: 'main' },
        { id: 'all', role: 'group' },
        { id: 'lead-bg', role: 'bg' },
      ],
      line: [{ start: 1000, end: 4000, value: 'Lead all echo' }],
      cueLine: [
        {
          index: 0,
          start: 2000,
          end: 3000,
          value: 'echo',
          agentId: 'lead-bg',
          cue: [
            { start: 2000, end: 3000, value: 'echo', byteStart: 0, byteEnd: 3 },
          ],
        },
        {
          index: 0,
          start: 1500,
          end: 2500,
          value: 'all',
          agentId: 'all',
          cue: [
            { start: 1500, end: 2500, value: 'all', byteStart: 0, byteEnd: 2 },
          ],
        },
        {
          index: 0,
          start: 1000,
          end: 2000,
          value: 'Lead',
          agentId: 'lead',
          cue: [
            { start: 1000, end: 2000, value: 'Lead', byteStart: 0, byteEnd: 3 },
          ],
        },
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
        {
          index: 0,
          start: 1000,
          end: 3000,
          value: '',
          cue: [
            { start: 1000, end: 2000, value: 'Lead' },
            { start: 2000, end: 3000, value: 'all' },
            { start: 3000, end: 3500, value: 'echo' },
          ],
        },
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

  it('estimates token windows when cue token timing is collapsed', () => {
    const lines = buildKaraokeLines({
      synced: true,
      line: [{ start: 1000, end: 5000, value: 'one two three four' }],
      cueLine: [
        {
          index: 0,
          start: 1000,
          end: 5000,
          value: 'one two three four',
          cue: [
            { start: 1000, end: 5000, value: 'one' },
            { start: 1000, end: 5000, value: 'two' },
            { start: 1000, end: 5000, value: 'three' },
            { start: 1000, end: 5000, value: 'four' },
          ],
        },
      ],
    })

    expect(resolveKaraokeTokenWindow(lines[0], 0)).toEqual({
      start: 1000,
      end: 2000,
    })
    expect(resolveKaraokeTokenWindow(lines[0], 3)).toEqual({
      start: 4000,
      end: 5000,
    })
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

  it('defaults available sidebar lyric layers on and keeps toggles local', () => {
    expect(
      resolveLyricsSidebarState({
        lyricsVisiblePreference: true,
        translationPreference: null,
        pronunciationPreference: null,
        hasMainLyric: true,
        hasTranslationLyric: true,
        hasPronunciationLyric: true,
      }),
    ).toEqual({
      lyricsVisible: true,
      showTranslation: true,
      showPronunciation: true,
    })

    expect(
      resolveLyricsSidebarState({
        lyricsVisiblePreference: 1,
        translationPreference: true,
        pronunciationPreference: true,
        hasTranslationLyric: 'yes',
        hasPronunciationLyric: undefined,
      }),
    ).toEqual({
      lyricsVisible: true,
      showTranslation: true,
      showPronunciation: false,
    })

    expect(
      resolveLyricsSidebarState({
        lyricsVisiblePreference: true,
        translationPreference: null,
        pronunciationPreference: null,
        hasTranslationLyric: false,
        hasPronunciationLyric: false,
      }),
    ).toEqual({
      lyricsVisible: true,
      showTranslation: false,
      showPronunciation: false,
    })

    expect(toggleLayerPreference(null, true)).toBe(false)
    expect(toggleLayerPreference(null, true, false)).toBe(true)
    expect(toggleLayerPreference(false, true)).toBe(true)
    expect(toggleLayerPreference(true, false)).toBe(false)
  })
})
