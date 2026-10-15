import { describe, it, expect } from 'vitest'
import { decode, naturalSize } from './thumbhash'

// Digests pinned from evanw/thumbhash's reference decoder (vendored at
// core/artwork/thumbhash/testdata/thumbhash.js), so any drift from it fails here.
const fnv = (bytes) => {
  let h = 2166136261 >>> 0
  for (const b of bytes) {
    h ^= b
    h = Math.imul(h, 16777619) >>> 0
  }
  return h.toString(16)
}

const REFERENCE = [
  ['alpha', 'JOiFBQ4nkIexh3p4iA8uB+lYhIeAh3d4dw==', 32, 32, '500141bc'],
  ['landscape', '3wcOFJpwh4eBh3d4iIePgAj3hw==', 32, 18, '18465aed'],
  ['portrait', '3/cNFBqBB4iId4d3d4iAjwj4hw==', 18, 32, '56500cc1'],
  ['solid', 'HoUBBwB4eHeHd3hweId3h3h4B2+Ih4gA', 32, 32, '39cc5c5'],
  ['square', 'H/gNBxpwh4dwd3eIiHd3iHeHeJ+dcH8I', 32, 32, '740d8afc'],
  ['tiny', 'HoU9tx4I9wiIh4hwj3CI+AiIcH/494cP', 32, 32, 'f2555987'],
]

describe('thumbhash decode', () => {
  it.each(REFERENCE)(
    'reproduces the reference pixels for %s',
    (_name, hash, w, h, digest) => {
      expect(fnv(decode(hash, w, h))).toEqual(digest)
    },
  )

  it.each(REFERENCE)(
    'reports the reference natural size for %s',
    (_name, hash, w, h) => {
      expect(naturalSize(hash)).toEqual({ width: w, height: h })
    },
  )

  it('decodes to any requested grid, not just the natural one', () => {
    const [, hash] = REFERENCE[4]
    const pixels = decode(hash, 8, 5)
    expect(pixels).toHaveLength(8 * 5 * 4)
    // Opaque hash: every alpha byte is saturated.
    for (let i = 3; i < pixels.length; i += 4) {
      expect(pixels[i]).toBe(255)
    }
  })

  it('carries alpha through for a hash that has it', () => {
    const [, hash, w, h] = REFERENCE[0]
    const pixels = decode(hash, w, h)
    const alphas = new Set()
    for (let i = 3; i < pixels.length; i += 4) {
      alphas.add(pixels[i])
    }
    expect(alphas.size).toBeGreaterThan(1)
  })

  it.each([
    ['empty', ''],
    ['too short to hold a header', 'AAAA'],
    ['not base64', '!!!not-a-thumbhash!!!'],
  ])('throws on a hash that is %s', (_name, hash) => {
    expect(() => decode(hash, 32, 32)).toThrow()
  })
})
