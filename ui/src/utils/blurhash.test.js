import { describe, it, expect } from 'vitest'
import { decode } from './blurhash'

// Pixel values pinned from the `blurhash` package before it was dropped, so any drift from the
// reference algorithm fails here.
const fnv = (bytes) => {
  let h = 2166136261 >>> 0
  for (const b of bytes) {
    h ^= b
    h = Math.imul(h, 16777619) >>> 0
  }
  return h.toString(16)
}

const HASHES = {
  square5x5: 'e2TI,c^_fQ^_fQ^_j@fQj@fQfQfQfQfQfQ^_j@fQj@fQfQfQfQfQfQ',
  classic4x3: 'LEHV6nWB2yk8pyo0adR*.7kCMdnj',
  wide6x4: 'U6PZfSi_.AyE_3t7t7R**0o#DgR4_3R*D%xt',
}

describe('blurhash decode', () => {
  it.each([
    [
      'square5x5',
      // prettier-ignore
      [255,255,0,255, 255,255,0,255, 255,255,0,255, 255,255,0,255,
       255,255,0,255, 253,254,0,255, 253,254,0,255, 253,254,0,255,
       255,255,0,255, 255,255,0,255, 255,255,0,255, 255,255,0,255],
    ],
    [
      'classic4x3',
      // prettier-ignore
      [135,164,177,255, 161,173,177,255, 181,180,171,255, 160,172,174,255,
       124,154,169,255, 148,148,154,255, 164,145,134,255, 146,152,155,255,
       124,144,154,255, 144,134,132,255, 163,130,104,255, 148,140,134,255],
    ],
    [
      'wide6x4',
      // prettier-ignore
      [231,230,228,255, 231,229,228,255, 233,232,230,255, 232,231,228,255,
       225,222,223,255, 218,210,204,255, 211,204,195,255, 217,214,211,255,
       225,223,222,255, 220,212,207,255, 221,215,206,255, 225,221,218,255],
    ],
  ])('reproduces the reference pixels for %s', (name, expected) => {
    expect(Array.from(decode(HASHES[name], 4, 3))).toEqual(expected)
  })

  it.each([
    ['square5x5', 32, 32, 'f0c527d8'],
    ['square5x5', 32, 18, 'aca15588'],
    ['classic4x3', 32, 32, '2097980e'],
    ['classic4x3', 32, 18, '1bb26307'],
    ['wide6x4', 32, 32, '4e23663e'],
    ['wide6x4', 32, 18, '2ef0355'],
  ])('reproduces the reference output for %s at %ix%i', (name, w, h, sum) => {
    const pixels = decode(HASHES[name], w, h)
    expect(pixels).toHaveLength(w * h * 4)
    expect(fnv(pixels)).toBe(sum)
  })

  it('makes every pixel opaque', () => {
    const pixels = decode(HASHES.classic4x3, 8, 8)
    for (let i = 3; i < pixels.length; i += 4) {
      expect(pixels[i]).toBe(255)
    }
  })

  it.each([
    ['empty', ''],
    ['too short', 'abc'],
    ['length mismatch', 'LEHV6nWB2yk8pyo0adR*.7kCMdn'],
    ['invalid character', 'LEHV6nWB2yk8pyo0adR*.7kCMdn\\'],
  ])('throws on a malformed hash (%s)', (name, hash) => {
    expect(() => decode(hash, 8, 8)).toThrow()
  })
})
