import { render } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { BlurHashCanvas } from './BlurHashCanvas'

describe('BlurHashCanvas', () => {
  // jsdom has no real 2D context; stub it so getContext doesn't hit its noisy "not implemented" path.
  let getContextSpy
  beforeEach(() => {
    getContextSpy = vi
      .spyOn(HTMLCanvasElement.prototype, 'getContext')
      .mockReturnValue({
        createImageData: (w, h) => ({ data: new Uint8ClampedArray(w * h * 4) }),
        putImageData: () => {},
      })
  })
  afterEach(() => {
    getContextSpy.mockRestore()
  })

  it('renders nothing without a hash', () => {
    const { container } = render(<BlurHashCanvas hash="" />)
    expect(container.querySelector('canvas')).toBeNull()
  })

  it('renders a canvas for a valid hash', () => {
    const { container } = render(
      <BlurHashCanvas hash="LEHV6nWB2yk8pyo0adR*.7kCMdnj" />,
    )
    expect(container.querySelector('canvas')).not.toBeNull()
  })

  it('renders a canvas without throwing on a malformed hash', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { container } = render(<BlurHashCanvas hash="!!!not-a-blurhash!!!" />)
    expect(container.querySelector('canvas')).not.toBeNull()
    spy.mockRestore()
  })
})
