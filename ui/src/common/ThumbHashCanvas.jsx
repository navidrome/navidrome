import { useEffect, useRef } from 'react'
import PropTypes from 'prop-types'
import { decode, naturalSize } from '../utils/thumbhash'

// A thumbhash carries no detail beyond a few dozen pixels; CSS upscales the canvas.
const DECODE_SIZE = 32

// bitmapSize prefers the artwork's true ratio, falling back to the aspect the hash itself carries,
// which is quantised to a ratio of small integers and so only approximates the image.
const bitmapSize = (hash, ratio) => {
  if (!(ratio > 0) || !Number.isFinite(ratio)) {
    try {
      return naturalSize(hash)
    } catch {
      return { width: DECODE_SIZE, height: DECODE_SIZE }
    }
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

export const ThumbHashCanvas = ({ hash, ratio, fit, className, style }) => {
  const canvasRef = useRef(null)
  const { width, height } = bitmapSize(hash, ratio)

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

ThumbHashCanvas.propTypes = {
  hash: PropTypes.string,
  // Aspect ratio (width / height) of the image this stands in for; the hash's own when omitted.
  ratio: PropTypes.number,
  fit: PropTypes.oneOf(['cover', 'contain']),
  className: PropTypes.string,
  style: PropTypes.object,
}
