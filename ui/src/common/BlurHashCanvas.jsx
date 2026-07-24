import { useEffect, useRef } from 'react'
import PropTypes from 'prop-types'
import { decode } from 'blurhash'

// A blurhash carries no detail beyond a few dozen pixels; CSS upscales the canvas.
const DECODE_SIZE = 32

export const BlurHashCanvas = ({ hash, className }) => {
  const canvasRef = useRef(null)

  useEffect(() => {
    if (!hash || !canvasRef.current) {
      return
    }
    try {
      const pixels = decode(hash, DECODE_SIZE, DECODE_SIZE)
      const ctx = canvasRef.current.getContext('2d')
      if (!ctx) {
        return
      }
      const imageData = ctx.createImageData(DECODE_SIZE, DECODE_SIZE)
      imageData.data.set(pixels)
      ctx.putImageData(imageData, 0, 0)
    } catch {
      // A malformed hash simply leaves the canvas transparent.
    }
  }, [hash])

  if (!hash) {
    return null
  }
  return (
    <canvas
      ref={canvasRef}
      width={DECODE_SIZE}
      height={DECODE_SIZE}
      className={className}
      aria-hidden="true"
    />
  )
}

BlurHashCanvas.propTypes = {
  hash: PropTypes.string,
  className: PropTypes.string,
}
