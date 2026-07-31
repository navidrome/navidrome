// ThumbHash decoder (https://github.com/evanw/thumbhash), kept byte-compatible with the reference:
// core/artwork/thumbhash/testdata/thumbhash.js pins the pixels the specs assert on.

const toBytes = (hash) => {
  if (typeof hash !== 'string' || hash === '') {
    throw new Error('thumbhash: empty hash')
  }
  const binary = atob(hash)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  if (bytes.length < 5) {
    throw new Error('thumbhash: hash too short')
  }
  return bytes
}

// header unpacks the fixed 5-byte prefix plus the optional alpha byte.
const header = (bytes) => {
  const header24 = bytes[0] | (bytes[1] << 8) | (bytes[2] << 16)
  const header16 = bytes[3] | (bytes[4] << 8)
  const hasAlpha = header24 >> 23 !== 0
  if (hasAlpha && bytes.length < 6) {
    throw new Error('thumbhash: hash too short for alpha')
  }
  const isLandscape = header16 >> 15 !== 0
  const alphaLimit = hasAlpha ? 5 : 7
  return {
    lDC: (header24 & 63) / 63,
    pDC: ((header24 >> 6) & 63) / 31.5 - 1,
    qDC: ((header24 >> 12) & 63) / 31.5 - 1,
    lScale: ((header24 >> 18) & 31) / 31,
    hasAlpha,
    pScale: ((header16 >> 3) & 63) / 63,
    qScale: ((header16 >> 9) & 63) / 63,
    lx: Math.max(3, isLandscape ? alphaLimit : header16 & 7),
    ly: Math.max(3, isLandscape ? header16 & 7 : alphaLimit),
    aDC: hasAlpha ? (bytes[5] & 15) / 15 : 1,
    aScale: hasAlpha ? bytes[5] >> 4 : 0,
  }
}

// naturalSize is the reference decoder's own output size: 32 on the long edge, at the aspect the
// hash approximates. Callers with the true dimensions should decode at those instead.
export const naturalSize = (hash) => {
  const { lx, ly } = header(toBytes(hash))
  const ratio = lx / ly
  return ratio > 1
    ? { width: 32, height: Math.round(32 / ratio) }
    : { width: Math.round(32 * ratio), height: 32 }
}

const cosTable = (n, size) => {
  const table = new Float64Array(n * size)
  for (let c = 0; c < n; c++) {
    for (let i = 0; i < size; i++) {
      table[c * size + i] = Math.cos(((Math.PI / size) * (i + 0.5) * c))
    }
  }
  return table
}

export const decode = (hash, width, height) => {
  const bytes = toBytes(hash)
  const h = header(bytes)
  if (!(width > 0) || !(height > 0)) {
    throw new Error('thumbhash: width and height must be positive')
  }

  const acStart = h.hasAlpha ? 6 : 5
  let acIndex = 0
  const channel = (nx, ny, scale) => {
    const ac = []
    for (let cy = 0; cy < ny; cy++) {
      for (let cx = cy ? 0 : 1; cx * ny < nx * (ny - cy); cx++) {
        const byte = bytes[acStart + (acIndex >> 1)] ?? 0
        ac.push((((byte >> ((acIndex++ & 1) << 2)) & 15) / 7.5 - 1) * scale)
      }
    }
    return ac
  }
  // The 1.25x chroma boost is the reference's quantisation compensation, not a free parameter.
  const lAC = channel(h.lx, h.ly, h.lScale)
  const pAC = channel(3, 3, h.pScale * 1.25)
  const qAC = channel(3, 3, h.qScale * 1.25)
  const aAC = h.hasAlpha ? channel(5, 5, h.aScale / 15) : []

  const nx = Math.max(h.lx, h.hasAlpha ? 5 : 3)
  const ny = Math.max(h.ly, h.hasAlpha ? 5 : 3)
  const fx = cosTable(nx, width)
  const fy = cosTable(ny, height)

  const pixels = new Uint8ClampedArray(width * height * 4)
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      let l = h.lDC
      let p = h.pDC
      let q = h.qDC
      let a = h.aDC

      for (let cy = 0, j = 0; cy < h.ly; cy++) {
        const fy2 = fy[cy * height + y] * 2
        for (let cx = cy ? 0 : 1; cx * h.ly < h.lx * (h.ly - cy); cx++, j++) {
          l += lAC[j] * fx[cx * width + x] * fy2
        }
      }
      for (let cy = 0, j = 0; cy < 3; cy++) {
        const fy2 = fy[cy * height + y] * 2
        for (let cx = cy ? 0 : 1; cx < 3 - cy; cx++, j++) {
          const f = fx[cx * width + x] * fy2
          p += pAC[j] * f
          q += qAC[j] * f
        }
      }
      if (h.hasAlpha) {
        for (let cy = 0, j = 0; cy < 5; cy++) {
          const fy2 = fy[cy * height + y] * 2
          for (let cx = cy ? 0 : 1; cx < 5 - cy; cx++, j++) {
            a += aAC[j] * fx[cx * width + x] * fy2
          }
        }
      }

      const b = l - (2 / 3) * p
      const r = (3 * l - b + q) / 2
      const g = r - q
      const idx = 4 * x + y * width * 4
      // Explicit floor: the reference writes into a Uint8Array, which truncates, while the
      // Uint8ClampedArray that createImageData needs would round.
      pixels[idx] = Math.floor(Math.max(0, 255 * Math.min(1, r)))
      pixels[idx + 1] = Math.floor(Math.max(0, 255 * Math.min(1, g)))
      pixels[idx + 2] = Math.floor(Math.max(0, 255 * Math.min(1, b)))
      pixels[idx + 3] = Math.floor(Math.max(0, 255 * Math.min(1, a)))
    }
  }
  return pixels
}
