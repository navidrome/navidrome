import {
  buildKaraokeLines,
  findLayerLineIndexForMain,
  getPreferredLyricLanguage,
  getActiveKaraokeState,
  hasStructuredLyricContent,
  pickStructuredLyric,
  resolveKaraokeTokenWindow,
  resolveLayerLineForMain,
  selectLyricLayers,
  structuredLyricToLrc,
  structuredLyricsToLrc,
} from './lyrics'

describe('lyrics helpers', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('prefers a lyric track that matches the locale', () => {
    const selected = pickStructuredLyric(
      [
        {
          lang: 'eng',
          synced: true,
          line: [{ start: 1000, value: 'English line' }],
        },
        {
          lang: 'pt-BR',
          synced: true,
          line: [{ start: 1000, value: 'Linha em portugues' }],
        },
      ],
      'pt-BR',
    )

    expect(selected.lang).toBe('pt-BR')
  })

  it('falls back to english when preferred locale is not available', () => {
    const selected = pickStructuredLyric(
      [
        {
          lang: 'eng',
          synced: true,
          line: [{ start: 1000, value: 'English line' }],
        },
        {
          lang: 'deu',
          synced: true,
          line: [{ start: 1000, value: 'Deutsche Zeile' }],
        },
      ],
      'pt-BR',
    )

    expect(selected.lang).toBe('eng')
  })

  it('falls back to first synced track when english is missing', () => {
    const selected = pickStructuredLyric(
      [
        {
          lang: 'jpn',
          synced: true,
          line: [{ start: 1000, value: 'Nihongo' }],
        },
        {
          lang: 'deu',
          synced: true,
          line: [{ start: 1000, value: 'Deutsch' }],
        },
      ],
      'pt-BR',
    )

    expect(selected.lang).toBe('jpn')
  })

  it('selects translation and pronunciation layers by kind', () => {
    const layers = selectLyricLayers(
      [
        {
          kind: 'main',
          lang: 'ja',
          synced: true,
          line: [{ start: 1000, value: 'こんにちは' }],
        },
        {
          kind: 'translation',
          lang: 'es',
          synced: true,
          line: [{ start: 1000, value: 'Hola' }],
        },
        {
          kind: 'pronunciation',
          lang: 'ja-Latn',
          synced: true,
          line: [{ start: 1000, value: 'konnichiwa' }],
        },
      ],
      'es-MX',
    )

    expect(layers.main.lang).toBe('ja')
    expect(layers.translation.lang).toBe('es')
    expect(layers.pronunciation.lang).toBe('ja-Latn')
  })

  it('treats missing kind as main for backward compatibility', () => {
    const layers = selectLyricLayers(
      [
        {
          lang: 'eng',
          synced: true,
          line: [{ start: 1000, value: 'Main' }],
        },
      ],
      'eng',
    )

    expect(layers.main.lang).toBe('eng')
    expect(layers.translation).toBeNull()
    expect(layers.pronunciation).toBeNull()
  })

  it('matches layer line by timing for the active main line', () => {
    const mainLines = [
      { index: 0, start: 1000, end: 1800, value: 'Line A', tokens: [] },
      { index: 1, start: 2000, end: 2800, value: 'Line B', tokens: [] },
    ]
    const layerLines = [
      { index: 0, start: 900, end: 1750, value: 'A2', tokens: [] },
      { index: 1, start: 2050, end: 2900, value: 'B2', tokens: [] },
    ]

    expect(findLayerLineIndexForMain(mainLines, layerLines, 1)).toBe(1)
    expect(resolveLayerLineForMain(mainLines, layerLines, 0).line.value).toBe(
      'A2',
    )
  })

  it('matches metadata layers by nearest timing even when indexes differ', () => {
    const mainLines = [
      { index: 0, start: 1000, end: 1800, value: 'Line A', tokens: [] },
      { index: 1, start: 2000, end: 2800, value: 'Line B', tokens: [] },
      { index: 2, start: 3000, end: 3800, value: 'Line C', tokens: [] },
    ]
    const layerLines = [
      { index: 2, start: 3020, end: 3820, value: 'C2', tokens: [] },
      { index: 0, start: 980, end: 1760, value: 'A2', tokens: [] },
      { index: 1, start: 2010, end: 2810, value: 'B2', tokens: [] },
    ]

    expect(findLayerLineIndexForMain(mainLines, layerLines, 1)).toBe(2)
    expect(resolveLayerLineForMain(mainLines, layerLines, 2).line.value).toBe(
      'C2',
    )
  })

  it('returns no layer match when the nearest line is too far in time', () => {
    const mainLines = [
      { index: 0, start: 1000, end: 1800, value: 'Line A', tokens: [] },
      { index: 1, start: 2000, end: 2800, value: 'Line B', tokens: [] },
    ]
    const layerLines = [
      { index: 0, start: 60000, end: 60800, value: 'Far line', tokens: [] },
    ]

    expect(findLayerLineIndexForMain(mainLines, layerLines, 1)).toBe(-1)
    expect(resolveLayerLineForMain(mainLines, layerLines, 1).line).toBeNull()
  })

  it('converts a structured lyric track to LRC', () => {
    const lrc = structuredLyricToLrc({
      lang: 'eng',
      synced: true,
      line: [
        { start: 18800, value: "We're no strangers to love" },
        { start: 22801, value: 'You know the rules and so do I' },
      ],
    })

    expect(lrc).toBe(
      "[00:18.80] We're no strangers to love\n[00:22.80] You know the rules and so do I\n",
    )
  })

  it('returns empty text when no synced lyrics are available', () => {
    const lrc = structuredLyricsToLrc(
      [{ lang: 'eng', synced: false, line: [{ value: 'Unsynced line' }] }],
      'eng',
    )

    expect(lrc).toBe('')
  })

  it('reads preferred language from localStorage first', () => {
    localStorage.setItem('locale', 'pt-BR')
    expect(getPreferredLyricLanguage()).toBe('pt-BR')
  })

  it('builds karaoke lines from tokenLine payload', () => {
    const lines = buildKaraokeLines({
      lang: 'eng',
      synced: true,
      line: [{ start: 1000, end: 3000, value: 'Hello world' }],
      tokenLine: [
        {
          index: 0,
          start: 1000,
          end: 3000,
          value: 'Hello world',
          token: [
            { start: 1000, end: 1500, value: 'Hello' },
            { start: 2000, end: 2500, value: 'world', role: 'x-bg' },
          ],
        },
      ],
    })

    expect(lines).toEqual([
      {
        index: 0,
        start: 1000,
        end: 3000,
        value: 'Hello world',
        tokens: [
          { start: 1000, end: 1500, value: 'Hello', role: '' },
          { start: 2000, end: 2500, value: 'world', role: 'x-bg' },
        ],
      },
    ])
  })

  it('sorts token timing by start to keep playback stable', () => {
    const lines = buildKaraokeLines({
      lang: 'eng',
      synced: true,
      line: [{ start: 1000, end: 3000, value: 'Hello world' }],
      tokenLine: [
        {
          index: 0,
          start: 1000,
          end: 3000,
          value: 'Hello world',
          token: [
            { start: 2000, end: 2500, value: 'world', role: '' },
            { start: 1000, end: 1500, value: 'Hello', role: '' },
          ],
        },
      ],
    })

    expect(lines[0].tokens.map((token) => token.value)).toEqual([
      'Hello',
      'world',
    ])
  })

  it('splits a single full-line token into synthetic word tokens', () => {
    const lines = buildKaraokeLines({
      lang: 'ko-Latn',
      synced: true,
      line: [{ start: 1000, end: 2000, value: 'Da-la-lun, dun' }],
      tokenLine: [
        {
          index: 0,
          start: 1000,
          end: 2000,
          value: 'Da-la-lun, dun',
          token: [{ start: 1000, end: 2000, value: 'Da-la-lun, dun' }],
        },
      ],
    })

    expect(lines).toHaveLength(1)
    expect(lines[0].tokens).toHaveLength(2)
    expect(lines[0].tokens[0].value).toBe('Da-la-lun, ')
    expect(lines[0].tokens[1].value).toBe('dun')

    const firstWindow = resolveKaraokeTokenWindow(lines[0], 0)
    const secondWindow = resolveKaraokeTokenWindow(lines[0], 1)

    expect(firstWindow.start).toBeCloseTo(1000)
    expect(firstWindow.end).toBeCloseTo(1500)
    expect(secondWindow.start).toBeCloseTo(1500)
    expect(secondWindow.end).toBeCloseTo(2000)
  })

  it('detects active line and token for karaoke timing', () => {
    const state = getActiveKaraokeState(
      [
        {
          index: 0,
          start: 1000,
          end: 3000,
          value: 'Hello world',
          tokens: [
            { start: 1000, end: 1500, value: 'Hello', role: '' },
            { start: 2000, end: 2500, value: 'world', role: '' },
          ],
        },
        {
          index: 1,
          start: 3500,
          end: 5000,
          value: 'Second line',
          tokens: [],
        },
      ],
      2200,
    )

    expect(state).toEqual({ lineIndex: 0, tokenIndex: 1 })
  })

  it('resolves token window fallback boundaries from neighboring tokens', () => {
    const line = {
      start: 1000,
      end: 3000,
      value: 'Hello world',
      tokens: [
        { start: 1200, value: 'Hello', role: '' },
        { start: 1800, value: 'world', role: '' },
      ],
    }

    expect(resolveKaraokeTokenWindow(line, 0)).toEqual({
      start: 1200,
      end: 1800,
    })
    expect(resolveKaraokeTokenWindow(line, 1)).toEqual({
      start: 1800,
      end: 3000,
    })
  })

  it('infers sequential token windows when token timings are missing', () => {
    const line = {
      start: 1000,
      end: 2000,
      value: 'A B C',
      tokens: [
        { value: 'A', role: '' },
        { value: 'B', role: '' },
        { value: 'C', role: '' },
      ],
    }

    const first = resolveKaraokeTokenWindow(line, 0)
    const second = resolveKaraokeTokenWindow(line, 1)
    const third = resolveKaraokeTokenWindow(line, 2)

    expect(first.start).toBeCloseTo(1000)
    expect(first.end).toBeCloseTo(1333.3333333333333)

    expect(second.start).toBeCloseTo(1333.3333333333333)
    expect(second.end).toBeCloseTo(1666.6666666666667)

    expect(third.start).toBeCloseTo(1666.6666666666667)
    expect(third.end).toBeCloseTo(2000)
  })

  it('falls back to sequential windows when token timings are collapsed', () => {
    const line = {
      start: 1000,
      end: 2000,
      value: 'A B C',
      tokens: [
        { start: 1000, end: 2000, value: 'A', role: '' },
        { start: 1000, end: 2000, value: 'B', role: '' },
        { start: 1000, end: 2000, value: 'C', role: '' },
      ],
    }

    const first = resolveKaraokeTokenWindow(line, 0)
    const second = resolveKaraokeTokenWindow(line, 1)
    const third = resolveKaraokeTokenWindow(line, 2)

    expect(first.start).toBeCloseTo(1000)
    expect(first.end).toBeCloseTo(1333.3333333333333)
    expect(second.start).toBeCloseTo(1333.3333333333333)
    expect(second.end).toBeCloseTo(1666.6666666666667)
    expect(third.start).toBeCloseTo(1666.6666666666667)
    expect(third.end).toBeCloseTo(2000)
  })

  it('keeps token selection stable near tight token boundaries', () => {
    const state = getActiveKaraokeState(
      [
        {
          index: 0,
          start: 1000,
          end: 2000,
          value: 'A B',
          tokens: [
            { start: 1000, end: 1100, value: 'A', role: '' },
            { start: 1110, end: 1300, value: 'B', role: '' },
          ],
        },
      ],
      1108,
    )

    expect(state).toEqual({ lineIndex: 0, tokenIndex: 0 })
  })

  it('reports structured lyric content when token timing exists', () => {
    expect(
      hasStructuredLyricContent({
        tokenLine: [{ token: [{ start: 100, value: 'a' }] }],
      }),
    ).toBe(true)
  })
})
