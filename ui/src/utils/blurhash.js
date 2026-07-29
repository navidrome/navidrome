const DIGITS =
  '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz#$%*+,-.:;=?@[]^_{|}~'

const decode83 = (str) => {
  let value = 0
  for (const char of str) {
    const digit = DIGITS.indexOf(char)
    if (digit < 0) {
      throw new Error(`blurhash: invalid character "${char}"`)
    }
    value = value * 83 + digit
  }
  return value
}

const sRGBToLinear = (value) => {
  const v = value / 255
  return v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
}

const linearTosRGB = (value) => {
  const v = Math.max(0, Math.min(1, value))
  return v <= 0.0031308
    ? Math.trunc(v * 12.92 * 255 + 0.5)
    : Math.trunc((1.055 * Math.pow(v, 1 / 2.4) - 0.055) * 255 + 0.5)
}

const signPow = (value, exp) =>
  (value < 0 ? -1 : 1) * Math.pow(Math.abs(value), exp)

const decodeDC = (value) => [
  sRGBToLinear(value >> 16),
  sRGBToLinear((value >> 8) & 255),
  sRGBToLinear(value & 255),
]

const decodeAC = (value, maxValue) => [
  signPow((Math.floor(value / 361) - 9) / 9, 2) * maxValue,
  signPow(((Math.floor(value / 19) % 19) - 9) / 9, 2) * maxValue,
  signPow(((value % 19) - 9) / 9, 2) * maxValue,
]

// Must stay byte-for-byte compatible with the `blurhash` package it replaced; the spec pins pixels.
export const decode = (hash, width, height) => {
  if (!hash || hash.length < 6) {
    throw new Error('blurhash: string must be at least 6 characters')
  }
  const sizeFlag = decode83(hash[0])
  const numY = Math.floor(sizeFlag / 9) + 1
  const numX = (sizeFlag % 9) + 1
  if (hash.length !== 4 + 2 * numX * numY) {
    throw new Error(
      `blurhash: length is ${hash.length} but it should be ${4 + 2 * numX * numY}`,
    )
  }

  const maxValue = (decode83(hash[1]) + 1) / 166
  const colors = new Array(numX * numY)
  colors[0] = decodeDC(decode83(hash.substring(2, 6)))
  for (let i = 1; i < colors.length; i++) {
    colors[i] = decodeAC(
      decode83(hash.substring(4 + i * 2, 6 + i * 2)),
      maxValue,
    )
  }

  // Tabulated rather than called per pixel per component: a 32x32 decode would otherwise make
  // tens of thousands of Math.cos calls, and a grid page mounts one of these per tile.
  const cosX = new Float64Array(width * numX)
  for (let i = 0; i < numX; i++) {
    for (let x = 0; x < width; x++) {
      cosX[i * width + x] = Math.cos((Math.PI * x * i) / width)
    }
  }
  const cosY = new Float64Array(height * numY)
  for (let j = 0; j < numY; j++) {
    for (let y = 0; y < height; y++) {
      cosY[j * height + y] = Math.cos((Math.PI * y * j) / height)
    }
  }

  const bytesPerRow = width * 4
  const pixels = new Uint8ClampedArray(bytesPerRow * height)
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      let r = 0
      let g = 0
      let b = 0
      for (let j = 0; j < numY; j++) {
        const basisY = cosY[j * height + y]
        for (let i = 0; i < numX; i++) {
          const basis = cosX[i * width + x] * basisY
          const color = colors[i + j * numX]
          r += color[0] * basis
          g += color[1] * basis
          b += color[2] * basis
        }
      }
      const idx = 4 * x + y * bytesPerRow
      pixels[idx] = linearTosRGB(r)
      pixels[idx + 1] = linearTosRGB(g)
      pixels[idx + 2] = linearTosRGB(b)
      pixels[idx + 3] = 255
    }
  }
  return pixels
}
