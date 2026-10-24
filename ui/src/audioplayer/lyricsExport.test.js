import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  buildLyricsFileName,
  downloadLyrics,
  lyricsFileExtension,
  serializeLyrics,
} from './lyricsExport'

const line = (value, start, end) => ({
  start,
  ...(end == null ? {} : { end }),
  value,
})

const cueLine = (index, value, cues) => ({ index, value, cue: cues })

const wordLyric = () => ({
  kind: 'main',
  lang: 'eng',
  synced: true,
  line: [line('Hello world', 8000, 11000)],
  cueLine: [
    cueLine(0, 'Hello world', [
      { start: 8000, value: 'Hello ' },
      { start: 9500, value: 'world' },
    ]),
  ],
})

describe('lyrics export', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('maps source formats to download extensions', () => {
    expect(lyricsFileExtension('ttml')).toBe('ttml')
    expect(lyricsFileExtension('ELRC')).toBe('lrc')
    expect(lyricsFileExtension('srt')).toBe('lrc')
    expect(lyricsFileExtension('plain')).toBe('txt')
    expect(lyricsFileExtension(undefined)).toBe('txt')
  })

  it('builds a sanitized Artist - Title (source).ext filename', () => {
    expect(
      buildLyricsFileName({
        artist: 'AC/DC: Live',
        title: 'Back? in Black',
        label: 'LRCLib',
        format: 'elrc',
      }),
    ).toBe('ACDC Live - Back in Black (LRCLib).lrc')

    expect(buildLyricsFileName({})).toBe(
      'Unknown artist - Unknown title (Lyrics).txt',
    )
  })

  it('serializes line and word timing to enhanced LRC', () => {
    expect(serializeLyrics(wordLyric(), 'elrc')).toBe(
      '[00:08.00]<00:08.00>Hello <00:09.50>world',
    )
  })

  it('folds the source offset into the exported timestamps', () => {
    const lyric = {
      kind: 'main',
      lang: 'eng',
      synced: true,
      offset: -500,
      line: [line('Hello world', 8000, 11000)],
      cueLine: [
        cueLine(0, 'Hello world', [
          { start: 8000, value: 'Hello ' },
          { start: 9500, value: 'world' },
        ]),
      ],
    }

    expect(serializeLyrics(lyric, 'elrc')).toBe(
      '[00:07.50]<00:07.50>Hello <00:09.00>world',
    )
    expect(serializeLyrics(lyric, 'ttml')).toContain(
      '<p begin="00:00:07.500" end="00:00:10.500">',
    )
  })

  it('serializes line timing to a plain LRC stamp per line', () => {
    const lyric = {
      kind: 'main',
      lang: 'eng',
      synced: true,
      line: [line('First line', 8000), line('Second line', 95010)],
    }

    expect(serializeLyrics(lyric, 'lrc')).toBe(
      '[00:08.00]First line\n[01:35.01]Second line',
    )
  })

  it('orders LRC lines chronologically, uses hour stamps, and clamps negatives', () => {
    const lyric = {
      kind: 'main',
      lang: 'eng',
      synced: false,
      line: [line('Long track', 6000500), line('Early line', -300)],
    }

    expect(serializeLyrics(lyric, 'lrc')).toBe(
      '[00:00.00]Early line\n[01:40:00.50]Long track',
    )
  })

  it('serializes a word-timed variant to TTML with escaped text', () => {
    const lyric = {
      kind: 'main',
      lang: 'eng',
      synced: true,
      line: [line('A & B', 8000, 11000)],
      cueLine: [
        cueLine(0, 'A & B', [{ start: 8000, end: 9500, value: 'A & B' }]),
      ],
    }

    expect(serializeLyrics(lyric, 'ttml')).toBe(
      [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" itunes:timing="Word" xml:lang="eng">',
        '<body><div>',
        '<p begin="00:00:08.000" end="00:00:11.000"><span begin="00:00:08.000" end="00:00:09.500">A &amp; B</span></p>',
        '</div></body></tt>',
      ].join('\n'),
    )
  })

  it('serializes untimed lyrics to TTML without a timing attribute', () => {
    const lyric = {
      kind: 'main',
      lang: 'xxx',
      synced: false,
      line: [line('Plain words')],
    }

    expect(serializeLyrics(lyric, 'ttml')).toBe(
      [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal">',
        '<body><div>',
        '<p>Plain words</p>',
        '</div></body></tt>',
      ].join('\n'),
    )
  })

  it('serializes plain lyrics as text lines', () => {
    const lyric = {
      kind: 'main',
      lang: 'eng',
      synced: false,
      line: [line('one'), line('two')],
    }

    expect(serializeLyrics(lyric, 'plain')).toBe('one\ntwo')
  })

  it('preserves UTF-8 gaps, punctuation, and final cue ends in downloads', () => {
    const lyric = {
      synced: true,
      line: [line('(你好， 世界!)', 1000, 3000)],
      cueLine: [
        cueLine(0, '(你好， 世界!)', [
          { start: 1000, end: 1400, value: '你好', byteStart: 1, byteEnd: 6 },
          { start: 2000, end: 2700, value: '世界', byteStart: 11, byteEnd: 16 },
        ]),
      ],
    }

    expect(serializeLyrics(lyric, 'elrc')).toBe(
      '[00:01.00]<00:01.00>(你好， <00:02.00>世界!)<00:02.70>',
    )
    const xml = serializeLyrics(lyric, 'ttml')
    const document = new DOMParser().parseFromString(xml, 'application/xml')
    expect(document.querySelector('parsererror')).toBeNull()
    expect(document.querySelector('p').textContent).toBe('(你好， 世界!)')
    expect(xml).toContain(
      '(<span begin="00:00:01.000" end="00:00:01.400">你好</span>， <span begin="00:00:02.000" end="00:00:02.700">世界</span>!)',
    )
  })

  it('preserves gaps when legacy cues have no byte ranges', () => {
    const lyric = wordLyric()
    lyric.line[0].value = 'Hello, world!'
    lyric.cueLine[0].value = 'Hello, world!'
    lyric.cueLine[0].cue[0].value = 'Hello'

    expect(serializeLyrics(lyric, 'lrc')).toBe(
      '[00:08.00]<00:08.00>Hello, <00:09.50>world!',
    )
  })

  it('keeps overlapping voice lanes separate instead of reversing inline timing', () => {
    const lyric = {
      synced: true,
      line: [line('Lead tail (echo)', 1000, 4000)],
      agents: [
        { id: 'lead', role: 'main', name: 'Lead & singer' },
        { id: '__nd_bg__|lead', role: 'bg', name: 'Echo' },
      ],
      cueLine: [
        {
          ...cueLine(0, 'Lead tail', [
            { start: 1000, end: 1500, value: 'Lead ' },
            { start: 3000, end: 3800, value: 'tail' },
          ]),
          agentId: 'lead',
          start: 1000,
          end: 4000,
        },
        {
          ...cueLine(0, '(echo)', [
            { start: 2000, end: 2500, value: '(echo)' },
          ]),
          agentId: '__nd_bg__|lead',
          start: 1000,
          end: 4000,
        },
      ],
    }

    expect(serializeLyrics(lyric, 'elrc')).toBe(
      '[00:01.00]<00:01.00>Lead <00:03.00>tail<00:03.80>\n[bg:<00:02.00>(echo)<00:02.50>]',
    )
    const xml = serializeLyrics(lyric, 'ttml')
    const document = new DOMParser().parseFromString(xml, 'application/xml')
    expect(document.querySelector('parsererror')).toBeNull()
    const paragraph = document.querySelector('p')
    expect(paragraph.textContent).toBe('Lead tail (echo)')
    const lanes = [...paragraph.children]
    expect(lanes.map((lane) => lane.getAttribute('ttm:agent'))).toEqual([
      'v1',
      'v2',
    ])
    expect(lanes[1].getAttribute('ttm:role')).toBe('x-bg')
    expect(lanes[0].textContent).toBe('Lead tail')
    expect(lanes[1].textContent).toBe('(echo)')
    expect(xml).toContain('<ttm:name>Lead &amp; singer</ttm:name>')
  })

  it.each(['lead', undefined])(
    'preserves line timing for separate TTML voices with main agent %s',
    (agentId) => {
      const lyric = {
        synced: true,
        line: [line('Lead Echo', 1000, 8000)],
        agents: [
          { id: 'lead', role: 'main' },
          { id: 'bg', role: 'bg' },
        ],
        cueLine: [
          {
            ...cueLine(0, 'Lead', []),
            agentId,
            start: 1000,
            end: 3000,
          },
          {
            ...cueLine(0, 'Echo', []),
            agentId: 'bg',
            start: 6000,
            end: 8000,
          },
        ],
      }
      const xml = serializeLyrics(lyric, 'ttml')
      const document = new DOMParser().parseFromString(xml, 'application/xml')
      expect(document.querySelector('parsererror')).toBeNull()
      const lanes = [...document.querySelector('p').children]
      expect(
        lanes.map((lane) => [
          lane.textContent,
          lane.getAttribute('begin'),
          lane.getAttribute('end'),
        ]),
      ).toEqual([
        ['Lead', '00:00:01.000', '00:00:03.000'],
        ['Echo', '00:00:06.000', '00:00:08.000'],
      ])
    },
  )

  it('exports background tags using the host LRC parser contract', () => {
    // model/lyrics_lrc_test.go covers this Main line / Okay / Next fixture.
    // A [bg:] row must remain immediately after its parent timestamped line,
    // with a final empty marker retaining the background cue's end time.
    const lyric = {
      synced: true,
      agents: [
        { id: 'main', role: 'main' },
        { id: '__nd_bg__|main', role: 'bg' },
      ],
      line: [line('Main line Okay', 52000), line('Next', 55000)],
      cueLine: [
        {
          ...cueLine(0, 'Main line', [
            { start: 52000, end: 52500, value: 'Main ' },
            { start: 52500, end: 53000, value: 'line' },
          ]),
          agentId: 'main',
        },
        {
          ...cueLine(0, 'Okay', [{ start: 53000, end: 53500, value: 'Okay' }]),
          agentId: '__nd_bg__|main',
        },
      ],
    }

    expect(serializeLyrics(lyric, 'elrc')).toBe(
      '[00:52.00]<00:52.00>Main <00:52.50>line<00:53.00>\n[bg:<00:53.00>Okay<00:53.50>]\n[00:55.00]Next',
    )
  })

  it('keeps a background with its parent when another voice is interleaved', () => {
    const lyric = {
      synced: true,
      agents: [
        { id: 'lead', role: 'main' },
        { id: 'guest', role: 'voice' },
        { id: 'bg', role: 'bg' },
      ],
      line: [line('Lead Guest Echo', 1000, 4000)],
      cueLine: [
        {
          ...cueLine(0, 'Lead', [{ start: 1000, end: 3000, value: 'Lead' }]),
          agentId: 'lead',
        },
        {
          ...cueLine(0, 'Guest', [{ start: 1500, end: 2500, value: 'Guest' }]),
          agentId: 'guest',
          start: 1500,
        },
        {
          ...cueLine(0, 'Echo', [{ start: 2000, end: 2800, value: 'Echo' }]),
          agentId: 'bg',
        },
      ],
    }

    expect(serializeLyrics(lyric, 'elrc')).toBe(
      '[00:01.00]<00:01.00>Lead<00:03.00>\n[bg:<00:02.00>Echo<00:02.80>]\n[00:01.50]<00:01.50>Guest<00:02.50>',
    )
  })

  it('preserves a timed background without foreground cues', () => {
    const lyric = {
      synced: true,
      agents: [{ id: 'bg', role: 'bg' }],
      line: [line('Echo', 1000, 2000)],
      cueLine: [
        {
          ...cueLine(0, 'Echo', [{ start: 1000, end: 2000, value: 'Echo' }]),
          agentId: 'bg',
        },
      ],
    }

    expect(serializeLyrics(lyric, 'elrc')).toBe(
      '[00:01.00]\n[bg:<00:01.00>Echo<00:02.00>]',
    )
  })

  it('keeps cue markers monotonic when byte ranges and cue times disagree', () => {
    const lyric = {
      synced: true,
      line: [line('first second', 1000, 3000)],
      cueLine: [
        cueLine(0, 'first second', [
          { start: 2000, end: 2500, value: 'first', byteStart: 0, byteEnd: 4 },
          {
            start: 1500,
            end: 1800,
            value: 'second',
            byteStart: 6,
            byteEnd: 11,
          },
        ]),
      ],
    }

    expect(serializeLyrics(lyric, 'elrc')).toBe(
      '[00:01.00]<00:02.00>first <00:02.00>second<00:02.00>',
    )
    const xml = serializeLyrics(lyric, 'ttml')
    expect(xml).toContain(
      '<span begin="00:00:02.000" end="00:00:02.500">first</span> <span begin="00:00:02.000" end="00:00:02.000">second</span>',
    )
  })

  it('downloads untimed LRC sources as plain text with a matching filename', () => {
    vi.useFakeTimers()
    const lyric = {
      synced: false,
      line: [line('Plain words'), line('Another line')],
    }
    const createObjectURL = vi.fn().mockReturnValue('blob:lyrics')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL })
    let filename
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(
      function () {
        filename = this.download
      },
    )

    expect(lyricsFileExtension('lrc', lyric)).toBe('txt')
    expect(serializeLyrics(lyric, 'lrc')).toBe('Plain words\nAnother line')
    downloadLyrics(lyric, {
      format: 'lrc',
      artist: 'Artist',
      title: 'Title',
      label: 'LRCLib',
    })

    expect(filename).toBe('Artist - Title (LRCLib).txt')
    expect(createObjectURL.mock.calls[0][0].type).toBe(
      'text/plain;charset=utf-8',
    )
    vi.runAllTimers()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:lyrics')
  })
})
