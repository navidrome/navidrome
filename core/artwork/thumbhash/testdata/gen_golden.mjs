// Produces golden.json from the vendored reference. Run: node gen_golden.mjs
import { readFileSync, writeFileSync, readdirSync } from 'fs'
import { inflateSync } from 'zlib'
import { rgbaToThumbHash } from './thumbhash.js'

// Minimal reader for the exact PNGs gen_fixtures.mjs writes: 8-bit RGBA, filter 0, single IDAT.
const readPNG = (buf) => {
  let w = 0, h = 0
  const idat = []
  for (let off = 8; off < buf.length; ) {
    const len = buf.readUInt32BE(off)
    const type = buf.toString('ascii', off + 4, off + 8)
    const data = buf.subarray(off + 8, off + 8 + len)
    if (type === 'IHDR') {
      w = data.readUInt32BE(0)
      h = data.readUInt32BE(4)
      if (data[8] !== 8 || data[9] !== 6) throw new Error('expected 8-bit RGBA')
    } else if (type === 'IDAT') idat.push(data)
    off += 12 + len
  }
  const raw = inflateSync(Buffer.concat(idat))
  const rgba = new Uint8Array(w * h * 4)
  for (let y = 0; y < h; y++) {
    if (raw[y * (w * 4 + 1)] !== 0) throw new Error('expected filter 0')
    for (let x = 0; x < w * 4; x++) rgba[y * w * 4 + x] = raw[y * (w * 4 + 1) + 1 + x]
  }
  return { w, h, rgba }
}

const dir = new URL('.', import.meta.url).pathname
const golden = {}
for (const name of readdirSync(dir).filter((f) => f.endsWith('.png')).sort()) {
  const { w, h, rgba } = readPNG(readFileSync(dir + name))
  golden[name] = Buffer.from(rgbaToThumbHash(w, h, rgba)).toString('base64')
}
writeFileSync(dir + 'golden.json', JSON.stringify(golden, null, 2) + '\n')
console.log(golden)
