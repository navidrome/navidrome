import { useEffect, useRef } from 'react'
import PropTypes from 'prop-types'
import { decode } from '../utils/blurhash'

// A blurhash carries no detail beyond a few dozen pixels; CSS upscales the canvas.
const DECODE_SIZE = 32

// bitmapSize shapes the decode target like the source image: a blurhash carries no aspect ratio,
// so a square decode stretched to the box distorts the blur and overpaints where the image won't reach.
const bitmapSize = (ratio) => {
  if (!(ratio > 0) || !Number.isFinite(ratio)) {
    return { width: DECODE_SIZE, height: DECODE_SIZE }
  }
  return ratio >= 1
    ? {
        width: DECODE_SIZE,
        height: Math.max(1, Math.round(DECODE_SIZE / ratio)),
      }
    : {
        width: Math.max(1, Math.round(DECODE_SIZE * ratio)),
        height: DECODE_SIZE,
      }
}

export const BlurHashCanvas = ({ hash, ratio, fit, className, style }) => {
  const canvasRef = useRef(null)
  const { width, height } = bitmapSize(ratio)

  useEffect(() => {
    if (!hash || !canvasRef.current) {
      return
    }
    const ctx = canvasRef.current.getContext('2d')
    if (!ctx) {
      return
    }
    // Clear first so a hash change that fails to decode never leaves a stale frame.
    ctx.clearRect(0, 0, width, height)
    try {
      const pixels = decode(hash, width, height)
      const imageData = ctx.createImageData(width, height)
      imageData.data.set(pixels)
      ctx.putImageData(imageData, 0, 0)
    } catch {
      // A malformed hash simply leaves the canvas blank.
    }
  }, [hash, width, height])

  if (!hash) {
    return null
  }
  return (
    <canvas
      ref={canvasRef}
      width={width}
      height={height}
      className={className}
      style={{ ...style, objectFit: fit }}
      aria-hidden="true"
    />
  )
}

BlurHashCanvas.propTypes = {
  hash: PropTypes.string,
  // Aspect ratio (width / height) of the image this stands in for; square when omitted or unusable.
  ratio: PropTypes.number,
  fit: PropTypes.oneOf(['cover', 'contain']),
  className: PropTypes.string,
  style: PropTypes.object,
}
