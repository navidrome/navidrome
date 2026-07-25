// Generates lossless PNG fixtures <=100px. Run: node gen_fixtures.mjs
import { writeFileSync } from 'fs'
import { deflateSync } from 'zlib'

const crcTable = Array.from({ length: 256 }, (_, n) => {
  let c = n
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
  return c >>> 0
})
const crc32 = (buf) => {
  let c = 0xffffffff
  for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8)
  return (c ^ 0xffffffff) >>> 0
}
const chunk = (type, data) => {
  const len = Buffer.alloc(4)
  len.writeUInt32BE(data.length)
  const body = Buffer.concat([Buffer.from(type, 'ascii'), data])
  const crc = Buffer.alloc(4)
  crc.writeUInt32BE(crc32(body))
  return Buffer.concat([len, body, crc])
}
const png = (w, h, rgba) => {
  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(w, 0)
  ihdr.writeUInt32BE(h, 4)
  ihdr[8] = 8 // bit depth
  ihdr[9] = 6 // truecolor + alpha
  const raw = Buffer.alloc(h * (w * 4 + 1))
  for (let y = 0; y < h; y++) {
    raw[y * (w * 4 + 1)] = 0 // filter: none
    for (let x = 0; x < w * 4; x++) raw[y * (w * 4 + 1) + 1 + x] = rgba[y * w * 4 + x]
  }
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk('IHDR', ihdr),
    chunk('IDAT', deflateSync(raw)),
    chunk('IEND', Buffer.alloc(0)),
  ])
}

const make = (w, h, fn) => {
  const rgba = new Uint8Array(w * h * 4)
  for (let y = 0; y < h; y++)
    for (let x = 0; x < w; x++) fn(rgba, (y * w + x) * 4, x, y, w, h)
  return png(w, h, rgba)
}

const gradient = (rgba, i, x, y, w, h) => {
  rgba[i] = Math.floor((255 * x) / w)
  rgba[i + 1] = Math.floor((255 * y) / h)
  rgba[i + 2] = Math.floor((255 * (x + y)) / (w + h))
  rgba[i + 3] = 255
}
const solid = (rgba, i) => {
  rgba[i] = 60
  rgba[i + 1] = 120
  rgba[i + 2] = 180
  rgba[i + 3] = 255
}
const alphaRamp = (rgba, i, x, y, w) => {
  rgba[i] = 200
  rgba[i + 1] = 50
  rgba[i + 2] = 90
  rgba[i + 3] = Math.floor((255 * x) / w)
}

const out = new URL('.', import.meta.url).pathname
writeFileSync(out + 'square.png', make(100, 100, gradient))
writeFileSync(out + 'landscape.png', make(100, 60, gradient))
writeFileSync(out + 'portrait.png', make(60, 100, gradient))
writeFileSync(out + 'alpha.png', make(80, 80, alphaRamp))
writeFileSync(out + 'solid.png', make(64, 64, solid))
writeFileSync(out + 'tiny.png', make(1, 1, solid))
console.log('fixtures written')
