import { render } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ThumbHashCanvas } from './ThumbHashCanvas'

// Golden hashes from core/artwork/thumbhash/testdata/golden.json.
const SQUARE = 'H/gNBxpwh4dwd3eIiHd3iHeHeJ+dcH8I'
const LANDSCAPE = '3wcOFJpwh4eBh3d4iIePgAj3hw=='

describe('ThumbHashCanvas', () => {
  // jsdom has no real 2D context; stub it (tracked) so specs can assert the draw path ran.
  let ctxMock
  let getContextSpy
  beforeEach(() => {
    ctxMock = {
      clearRect: vi.fn(),
      createImageData: vi.fn((w, h) => ({
        data: new Uint8ClampedArray(w * h * 4),
      })),
      putImageData: vi.fn(),
    }
    getContextSpy = vi
      .spyOn(HTMLCanvasElement.prototype, 'getContext')
      .mockReturnValue(ctxMock)
  })
  afterEach(() => {
    getContextSpy.mockRestore()
  })

  it('renders nothing without a hash', () => {
    const { container } = render(<ThumbHashCanvas hash="" />)
    expect(container.querySelector('canvas')).toBeNull()
  })

  it('decodes a valid hash and draws non-trivial pixel data', () => {
    const { container } = render(<ThumbHashCanvas hash={SQUARE} />)
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(ctxMock.putImageData).toHaveBeenCalledTimes(1)
    const [imageData] = ctxMock.putImageData.mock.calls[0]
    expect(imageData.data.some((byte) => byte !== 0)).toBe(true)
  })

  it('decodes into a bitmap shaped like the image, so the blur is not distorted', () => {
    const { container } = render(
      <ThumbHashCanvas hash={SQUARE} ratio={1200 / 800} />,
    )
    expect(ctxMock.createImageData).toHaveBeenCalledWith(32, 21)
    const canvas = container.querySelector('canvas')
    expect(canvas.width).toBe(32)
    expect(canvas.height).toBe(21)
  })

  it('shapes a portrait ratio the other way round', () => {
    render(<ThumbHashCanvas hash={SQUARE} ratio={0.5} />)
    expect(ctxMock.createImageData).toHaveBeenCalledWith(16, 32)
  })

  it('never collapses an extreme ratio to a zero-sized bitmap', () => {
    render(<ThumbHashCanvas hash={SQUARE} ratio={200} />)
    expect(ctxMock.createImageData).toHaveBeenCalledWith(32, 1)
  })

  // Unlike a blurhash, a thumbhash carries its own approximate aspect, so an unknown ratio
  // falls back to that rather than to a square.
  it('falls back to the hash own aspect when the ratio is unknown', () => {
    render(<ThumbHashCanvas hash={LANDSCAPE} ratio={0} />)
    expect(ctxMock.createImageData).toHaveBeenCalledWith(32, 18)
  })

  it('applies the object-fit it is given, so it lands where the image will', () => {
    const { container } = render(
      <ThumbHashCanvas hash={SQUARE} ratio={1.5} fit="contain" />,
    )
    expect(container.querySelector('canvas').style.objectFit).toBe('contain')
  })

  it('renders a canvas without throwing on a malformed hash, and draws nothing', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { container } = render(
      <ThumbHashCanvas hash="!!!not-a-thumbhash!!!" />,
    )
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(ctxMock.putImageData).not.toHaveBeenCalled()
    spy.mockRestore()
  })

  it('clears the canvas when a hash change fails to decode', () => {
    const { rerender } = render(<ThumbHashCanvas hash={SQUARE} />)
    expect(ctxMock.putImageData).toHaveBeenCalledTimes(1)

    rerender(<ThumbHashCanvas hash="!!!not-a-thumbhash!!!" />)

    expect(ctxMock.clearRect).toHaveBeenCalledTimes(2)
    expect(ctxMock.putImageData).toHaveBeenCalledTimes(1)
  })
})
