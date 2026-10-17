import { describe, expect, it } from 'vitest'
import {
  buildLyricsFileName,
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

  it('switches to hh:mm:ss stamps past 99 minutes and clamps negatives', () => {
    const lyric = {
      kind: 'main',
      lang: 'eng',
      synced: false,
      line: [line('Long track', 6000500), line('Early line', -300)],
    }

    expect(serializeLyrics(lyric, 'lrc')).toBe(
      '[01:40:00.50]Long track\n[00:00.00]Early line',
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
})
