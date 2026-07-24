import { render } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { BlurHashCanvas } from './BlurHashCanvas'

describe('BlurHashCanvas', () => {
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
    const { container } = render(<BlurHashCanvas hash="" />)
    expect(container.querySelector('canvas')).toBeNull()
  })

  it('decodes a valid hash and draws non-trivial pixel data', () => {
    const { container } = render(
      <BlurHashCanvas hash="LEHV6nWB2yk8pyo0adR*.7kCMdnj" />,
    )
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(ctxMock.createImageData).toHaveBeenCalledWith(32, 32)
    expect(ctxMock.putImageData).toHaveBeenCalledTimes(1)
    const [imageData] = ctxMock.putImageData.mock.calls[0]
    expect(imageData.data.some((byte) => byte !== 0)).toBe(true)
  })

  it('renders a canvas without throwing on a malformed hash, and draws nothing', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { container } = render(<BlurHashCanvas hash="!!!not-a-blurhash!!!" />)
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(ctxMock.putImageData).not.toHaveBeenCalled()
    spy.mockRestore()
  })

  it('clears the canvas when a hash change fails to decode', () => {
    const { rerender } = render(
      <BlurHashCanvas hash="LEHV6nWB2yk8pyo0adR*.7kCMdnj" />,
    )
    expect(ctxMock.putImageData).toHaveBeenCalledTimes(1)

    rerender(<BlurHashCanvas hash="!!!not-a-blurhash!!!" />)

    expect(ctxMock.clearRect).toHaveBeenCalledTimes(2)
    // No new pixels drawn after the clear, so the stale frame stays gone.
    expect(ctxMock.putImageData).toHaveBeenCalledTimes(1)
  })
})
