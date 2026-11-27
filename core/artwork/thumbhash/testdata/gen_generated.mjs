// Produces generated.json: ThumbHashes of synthetic images, from the vendored reference.
// The pixels are a pure function of their index, so Go rebuilds them byte-for-byte from the same
// formula and only the hashes need committing. Run: node gen_generated.mjs
import { writeFileSync } from 'fs'
import { rgbaToThumbHash } from './thumbhash.js'

const COUNT = 300

// mix is a stateless 32-bit finaliser; Go's mix in thumbhash_test.go must match it exactly.
const mix = (n) => {
  n = Math.imul(n ^ (n >>> 16), 2246822507) >>> 0
  n = Math.imul(n ^ (n >>> 13), 3266489909) >>> 0
  return (n ^ (n >>> 16)) >>> 0
}

const out = []
for (let i = 0; i < COUNT; i++) {
  const w = 1 + (mix(i * 3 + 1) % 100)
  const h = 1 + (mix(i * 3 + 2) % 100)
  const rgba = new Uint8Array(w * h * 4)
  for (let k = 0; k < rgba.length; k++) rgba[k] = mix(i * 1000003 + k) & 255
  // Random alpha is opaque essentially never, so half are forced opaque to exercise the 7x7
  // no-alpha layout as often as the 5x5-plus-alpha one.
  if (i % 2 === 0) for (let k = 3; k < rgba.length; k += 4) rgba[k] = 255
  out.push({ w, h, hash: Buffer.from(rgbaToThumbHash(w, h, rgba)).toString('base64') })
}
writeFileSync(new URL('.', import.meta.url).pathname + 'generated.json', JSON.stringify(out) + '\n')
console.log(`${out.length} vectors; first ${out[0].w}x${out[0].h} ${out[0].hash}`)
