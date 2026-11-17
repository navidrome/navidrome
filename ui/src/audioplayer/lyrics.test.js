import { describe, expect, it, vi } from 'vitest'
import {
  buildKaraokeLines,
  clampLyricsOffset,
  collectLyricVariants,
  formatLyricsOffset,
  findLyricVariant,
  getPreferredLyricLanguage,
  hasNudgeableTiming,
  hasStructuredLyricContent,
  selectLyricLayers,
  stepLyricsOffset,
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

describe('lyrics offset', () => {
  it('clamps the offset to the limit', () => {
    expect(clampLyricsOffset(0)).toBe(0)
    expect(clampLyricsOffset(4999)).toBe(4999)
    expect(clampLyricsOffset(5001)).toBe(5000)
    expect(clampLyricsOffset(-5001)).toBe(-5000)
    expect(clampLyricsOffset(Number.NaN)).toBe(0)
  })

  it('steps by the given delta and clamps at the limit', () => {
    let offset = 3000
    for (let step = 0; step < 5; step += 1) {
      offset = stepLyricsOffset(offset, 500)
    }
    expect(offset).toBe(5000)
  })

  it('formats the offset for display', () => {
    expect(formatLyricsOffset(-50)).toBe('-0.05 s')
    expect(formatLyricsOffset(0)).toBe('0.00 s')
    expect(formatLyricsOffset(1250)).toBe('+1.25 s')
  })

  it('reports whether a lyric can take a timing nudge', () => {
    expect(
      hasNudgeableTiming({
        kind: 'main',
        lang: 'en',
        synced: false,
        line: [{ value: 'plain words' }],
      }),
    ).toBe(false)
    expect(hasNudgeableTiming(structured('main', 'en', 'timed line'))).toBe(
      true,
    )
    expect(
      hasNudgeableTiming({
        kind: 'main',
        lang: 'en',
        synced: false,
        line: [{ value: 'word timed' }],
        cueLine: [
          { index: 0, value: 'word timed', cue: [{ start: 0, value: 'word' }] },
        ],
      }),
    ).toBe(true)
  })

  it('applies the user offset on top of the source offset', () => {
    const lines = buildKaraokeLines(
      { ...structured('main', 'en', 'Hello'), offset: 500 },
      -300,
    )

    expect(lines[0].start).toBe(200)
  })

  it('shifts cue word starts by the user offset', () => {
    const lines = buildKaraokeLines(
      {
        kind: 'main',
        lang: 'en',
        synced: true,
        line: [timed('Hello world', 8000, 11000)],
        cueLine: [
          cueLine(0, 'Hello world', 8000, 11000, [
            { start: 8000, value: 'Hello ' },
            { start: 9500, value: 'world' },
          ]),
        ],
      },
      -500,
    )

    expect(lines[0].start).toBe(7500)
    expect(lines[0].tokens[0].start).toBe(7500)
    expect(lines[0].tokens[1].start).toBe(9000)
  })
})

describe('lyrics variants', () => {
  it('keeps the same provider in different plugins independently selectable', () => {
    const entries = ['First Plugin', 'Second Plugin'].map((name) => ({
      ...structured('main', 'eng', name),
      source: { type: 'plugin', name, provider: 'LRCLib', format: 'lrc' },
    }))
    const variants = collectLyricVariants(entries)
    expect(variants).toHaveLength(2)
    expect(
      selectLyricLayers(entries, 'en', variants[1].key).main.line[0].value,
    ).toBe('Second Plugin')
  })

  it('keys plugin variants by installation id instead of display name', () => {
    const source = (pluginId) => ({
      type: 'plugin',
      name: 'Same Display Name',
      pluginId,
      provider: 'LRCLib',
      format: 'lrc',
    })
    const entries = [
      {
        ...structured('main', 'eng', 'First install'),
        source: source('zz-first'),
      },
      {
        ...structured('main', 'eng', 'Second install'),
        source: source('zz-second'),
      },
    ]
    const variants = collectLyricVariants(entries)
    expect(variants.map((variant) => variant.key)).toEqual([
      'plugin:zz-first:lrclib',
      'plugin:zz-second:lrclib',
    ])
    expect(
      selectLyricLayers(entries, 'en', variants[1].key).main.line[0].value,
    ).toBe('Second install')
  })

  it('associates auxiliary tracks by recording identity rather than selection token', () => {
    const source = {
      type: 'plugin',
      name: 'Same Display Name',
      pluginId: 'nd-better-lyrics',
      provider: 'LRCLib',
    }
    const variant = (role, language, selection, extra = {}) => ({
      version: 1,
      instance: 'lrclib:0123456789abcdef0123456789abcdef',
      recording: 'rec-1',
      role,
      language,
      format: 'lrc',
      ...extra,
    })
    const main = {
      ...structured('main', 'en', 'Main words'),
      source,
      selection: 'sel-main',
      variant: variant('main', 'en', 'sel-main'),
    }
    const translation = {
      ...structured('translation', 'es', 'Hola'),
      source,
      selection: 'sel-translation',
      variant: variant('translation', 'es', 'sel-translation'),
    }
    const pronunciation = {
      ...structured('pronunciation', 'en-Latn', 'men'),
      source,
      selection: 'sel-pronunciation',
      variant: variant('pronunciation', 'en-Latn', 'sel-pronunciation'),
    }

    const variants = collectLyricVariants([main, translation, pronunciation])
    expect(variants).toHaveLength(1)
    const layers = selectLyricLayers(
      [main, translation, pronunciation],
      'es',
      variants[0].key,
    )
    expect(layers.main.line[0].value).toBe('Main words')
    expect(layers.translation.line[0].value).toBe('Hola')
    expect(layers.pronunciation.line[0].value).toBe('men')
  })

  it('does not leak auxiliary tracks across recordings or plugin installations', () => {
    const source = (pluginId) => ({
      type: 'plugin',
      name: 'Same Display Name',
      pluginId,
      provider: 'LRCLib',
    })
    const variant = (role, recording, selection) => ({
      version: 1,
      instance: 'lrclib:0123456789abcdef0123456789abcdef',
      recording,
      role,
      language: 'es',
      format: 'lrc',
    })
    const main = {
      ...structured('main', 'en', 'First recording'),
      source: source('nd-better-lyrics'),
      selection: 'sel-first',
      variant: variant('main', 'rec-1', 'sel-first'),
    }
    const otherMain = {
      ...structured('main', 'en', 'Second recording'),
      source: source('nd-better-lyrics'),
      selection: 'sel-second',
      variant: variant('main', 'rec-2', 'sel-second'),
    }
    const crossRecording = {
      ...structured('translation', 'es', 'Otra grabación'),
      source: source('nd-better-lyrics'),
      selection: 'sel-other',
      variant: variant('translation', 'rec-2', 'sel-other'),
    }
    const crossInstall = {
      ...structured('translation', 'es', 'Otra instalación'),
      source: source('another-install'),
      selection: 'sel-install',
      variant: variant('translation', 'rec-1', 'sel-install'),
    }

    const variants = collectLyricVariants([main, otherMain])
    expect(
      selectLyricLayers(
        [main, otherMain, crossRecording],
        'es',
        variants[1].key,
      ).translation,
    ).toBe(crossRecording)
    const isolated = selectLyricLayers(
      [main, crossInstall],
      'es',
      variants[0].key,
    )
    expect(isolated.translation).toBeNull()
  })

  it('keeps sidecar formats and their translation layers separate', () => {
    const lrc = { type: 'sidecar', format: 'lrc' }
    const ttml = { type: 'sidecar', format: 'ttml' }
    const lyrics = [
      { ...structured('main', 'eng', 'Local LRC'), source: lrc },
      { ...structured('main', 'eng', 'Local TTML'), source: ttml },
      { ...structured('translation', 'eng', 'TTML translation'), source: ttml },
    ]
    const variants = collectLyricVariants(lyrics)
    expect(variants).toHaveLength(2)
    expect(
      selectLyricLayers(lyrics, 'en', variants[0].key).translation,
    ).toBeNull()
    const selected = selectLyricLayers(lyrics, 'en', variants[1].key)
    expect(selected.main.line[0].value).toBe('Local TTML')
    expect(selected.translation.line[0].value).toBe('TTML translation')
  })

  it('groups main entries from the same source into one variant', () => {
    const lyrics = [
      {
        ...structured('main', 'eng', 'Hello'),
        source: { type: 'embedded' },
      },
      {
        ...structured('main', 'jpn', 'こんにちは'),
        source: { type: 'embedded' },
      },
    ]

    const variants = collectLyricVariants(lyrics)

    expect(variants).toHaveLength(1)
    expect(variants[0].key).toBe('embedded')
  })

  it('lists plugin variants in response order with their sync levels', () => {
    const wordEntry = {
      kind: 'main',
      lang: 'eng',
      synced: true,
      line: [timed('word line', 0)],
      cueLine: [
        cueLine(0, 'word line', 0, 1000, [{ start: 0, value: 'word line' }]),
      ],
      source: {
        type: 'plugin',
        name: 'Better Lyrics',
        provider: 'NetEase',
        format: 'elrc',
      },
    }
    const lineEntry = {
      ...structured('main', 'eng', 'line line'),
      source: {
        type: 'plugin',
        name: 'Better Lyrics',
        provider: 'LRCLib',
        format: 'lrc',
      },
    }
    const plainEntry = {
      kind: 'main',
      lang: 'eng',
      synced: false,
      line: [{ value: 'plain words' }],
      source: { type: 'plugin', provider: 'Genius', format: 'plain' },
    }

    expect(
      collectLyricVariants([wordEntry, lineEntry, plainEntry]).map((v) => [
        v.key,
        v.sync,
      ]),
    ).toEqual([
      ['plugin:better%20lyrics:netease', 'word'],
      ['plugin:better%20lyrics:lrclib', 'line'],
      ['genius', 'plain'],
    ])
  })

  it('selects the requested variant and falls back to the winner', () => {
    const netease = {
      ...structured('main', 'eng', 'NetEase line'),
      source: { provider: 'NetEase', format: 'elrc' },
    }
    const lrclib = {
      ...structured('main', 'eng', 'LRCLib line'),
      source: { provider: 'LRCLib', format: 'lrc' },
    }
    const lyrics = [netease, lrclib]

    expect(selectLyricLayers(lyrics, 'en').main.line[0].value).toBe(
      'NetEase line',
    )
    expect(selectLyricLayers(lyrics, 'en', 'lrclib').main.line[0].value).toBe(
      'LRCLib line',
    )
    expect(selectLyricLayers(lyrics, 'en', 'gone').main.line[0].value).toBe(
      'NetEase line',
    )
  })

  it('keeps the language pick inside the selected variant', () => {
    const english = {
      ...structured('main', 'eng', 'English words'),
      source: { provider: 'NetEase' },
    }
    const japanese = {
      ...structured('main', 'jpn', '日本語の歌詞'),
      source: { provider: 'NetEase' },
    }

    expect(
      selectLyricLayers([english, japanese], 'ja').main.line[0].value,
    ).toBe('日本語の歌詞')
    expect(
      selectLyricLayers([english, japanese], 'en').main.line[0].value,
    ).toBe('English words')
  })

  it('scopes translation layers to the selected source', () => {
    const neteaseMain = {
      ...structured('main', 'eng', 'NetEase main'),
      source: { provider: 'NetEase' },
    }
    const geniusMain = {
      ...structured('main', 'eng', 'Genius main'),
      source: { provider: 'Genius' },
    }
    const neteaseTranslation = {
      ...structured('translation', 'es', 'Traducción NetEase'),
      source: { provider: 'NetEase' },
    }
    const geniusTranslation = {
      ...structured('translation', 'fr', 'Traduction Genius'),
      source: { provider: 'Genius' },
    }

    const scoped = selectLyricLayers(
      [neteaseMain, geniusMain, neteaseTranslation, geniusTranslation],
      'en',
      'genius',
    )
    expect(scoped.main.line[0].value).toBe('Genius main')
    expect(scoped.translation.line[0].value).toBe('Traduction Genius')

    const fallback = selectLyricLayers(
      [neteaseMain, geniusMain, geniusTranslation],
      'en',
      'netease',
    )
    expect(fallback.main.line[0].value).toBe('NetEase main')
    expect(fallback.translation).toBeNull()
  })

  it('uses untagged legacy auxiliary tracks without mixing explicit providers', () => {
    const main = {
      ...structured('main', 'ja', '日本語'),
      source: { type: 'plugin', provider: 'NetEase' },
    }
    const translation = structured('translation', 'en', 'Japanese')
    const pronunciation = structured('pronunciation', 'ja-Latn', 'nihongo')
    const otherProvider = {
      ...structured('pronunciation', 'ja-Latn', 'other pronunciation'),
      source: { type: 'plugin', provider: 'LyricsPlus' },
    }
    const layers = selectLyricLayers(
      [main, translation, otherProvider, pronunciation],
      'en',
      'netease',
    )

    expect(layers.translation).toBe(translation)
    expect(layers.pronunciation).toBe(pronunciation)
    expect(
      selectLyricLayers([main, otherProvider], 'en').pronunciation,
    ).toBeNull()
  })
})

describe('recording match selection', () => {
  const matched = (value, status, selection, extra = {}) => ({
    kind: 'main',
    lang: 'en',
    synced: true,
    line: [{ start: 0, value }],
    selection,
    match: { status, autoSaveEligible: status === 'validated' },
    ...extra,
  })

  it('never auto-selects an ambiguous suggestion', () => {
    const layers = selectLyricLayers(
      [
        matched('Wrong live words', 'ambiguous', 'sel-live'),
        matched('Right studio words', 'validated', 'sel-studio'),
      ],
      'en',
    )
    expect(layers.main.line[0].value).toBe('Right studio words')
  })

  it('does not auto-select when only ambiguous suggestions exist', () => {
    const layers = selectLyricLayers(
      [matched('Wrong live words', 'ambiguous', 'sel-live')],
      'en',
    )
    expect(layers.main).toBeNull()
  })

  it('allows an ambiguous suggestion to be selected explicitly', () => {
    const entries = [matched('Wrong live words', 'ambiguous', 'sel-live')]
    const layers = selectLyricLayers(entries, 'en', 'selection:sel-live')
    expect(layers.main.line[0].value).toBe('Wrong live words')
    expect(
      findLyricVariant(collectLyricVariants(entries), 'selection:sel-live')
        ?.suggestion,
    ).toBe(true)
  })

  it('keeps two selections from one source individually addressable', () => {
    const variants = collectLyricVariants([
      matched('First recording', 'validated', 'sel-a'),
      matched('Second recording', 'ambiguous', 'sel-b'),
    ])
    expect(variants).toHaveLength(2)
    expect(variants[0].key).toBe('selection:sel-a')
    expect(variants[1].key).toBe('selection:sel-b')
  })
})

describe('selection preference compatibility', () => {
  const main = {
    kind: 'main',
    lang: 'en',
    synced: true,
    line: [{ start: 0, value: 'selected main' }],
    selection: 'sel-main',
    match: { status: 'validated', autoSaveEligible: true },
    source: { type: 'plugin', name: 'Better Lyrics', provider: 'LRCLib' },
  }

  it('resolves a preference saved with the old composite source key', () => {
    const oldKey = 'plugin:better%20lyrics:lrclib'
    expect(findLyricVariant(collectLyricVariants([main]), oldKey)).toBeTruthy()
  })

  it('scopes an auxiliary layer that shares the main selection', () => {
    const translation = {
      kind: 'translation',
      lang: 'es',
      synced: true,
      line: [{ start: 0, value: 'traducción' }],
      selection: 'sel-main',
      source: { type: 'plugin', name: 'Better Lyrics', provider: 'LRCLib' },
    }
    const layers = selectLyricLayers(
      [main, translation],
      'en',
      'selection:sel-main',
    )
    expect(layers.main.line[0].value).toBe('selected main')
    expect(layers.translation.line[0].value).toBe('traducción')
  })
})

describe('invalid match status auto-selection', () => {
  const entry = (value, status) => ({
    kind: 'main',
    lang: 'en',
    synced: true,
    line: [{ start: 0, value }],
    match: { status, autoSaveEligible: false },
  })

  it('skips a mismatch before a playable result', () => {
    const layers = selectLyricLayers(
      [entry('wrong', 'mismatch'), entry('right', 'validated')],
      'en',
    )
    expect(layers.main.line[0].value).toBe('right')
  })

  it('does not auto-select when only mismatches exist', () => {
    expect(selectLyricLayers([entry('wrong', 'mismatch')], 'en').main).toBe(
      null,
    )
  })

  it('does not treat an unknown assessed status as playable', () => {
    expect(selectLyricLayers([entry('unknown', 'pending')], 'en').main).toBe(
      null,
    )
  })
})

describe('invalid assessed match statuses', () => {
  const matched = (status) => ({
    kind: 'main',
    lang: 'en',
    synced: true,
    line: [{ start: 0, value: 'assessed entry' }],
    match: status === undefined ? {} : { status },
  })

  it('fails closed for a present match object with no status', () => {
    expect(selectLyricLayers([matched()], 'en').main).toBeNull()
  })

  it('fails closed for a present match object with an empty status', () => {
    expect(selectLyricLayers([matched('')], 'en').main).toBeNull()
  })

  it('fails closed for an unknown assessed status while keeping legacy absence compatible', () => {
    expect(selectLyricLayers([matched('pending')], 'en').main).toBeNull()
    const legacy = {
      kind: 'main',
      lang: 'en',
      synced: true,
      line: [{ start: 0, value: 'legacy entry' }],
    }
    expect(selectLyricLayers([legacy], 'en').main.line[0].value).toBe(
      'legacy entry',
    )
  })
})

describe('suggestion-only auxiliary layers', () => {
  it('returns no main, translation or pronunciation layers for an unselected suggestion', () => {
    const suggestion = {
      kind: 'main',
      lang: 'en',
      synced: true,
      line: [{ start: 0, value: 'suggested main' }],
      match: { status: 'ambiguous' },
    }
    const translation = {
      kind: 'translation',
      lang: 'es',
      synced: true,
      line: [{ start: 0, value: 'suggested translation' }],
      match: { status: 'ambiguous' },
    }
    const pronunciation = {
      kind: 'pronunciation',
      lang: 'en',
      synced: true,
      line: [{ start: 0, value: 'suggested pronunciation' }],
      match: { status: 'ambiguous' },
    }
    expect(
      selectLyricLayers([suggestion, translation, pronunciation], 'en'),
    ).toEqual({ main: null, translation: null, pronunciation: null })
  })
})
