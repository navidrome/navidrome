import { render, fireEvent, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('./useImageUrl', () => ({ useImageUrl: vi.fn() }))
vi.mock('../subsonic', () => ({
  default: { getCoverArtUrl: () => '/rest/getCoverArt?id=al-1' },
}))
vi.mock('../config', () => ({ default: { uiCoverArtSize: 300 } }))

import { useImageUrl } from './useImageUrl'
import { Artwork } from './Artwork'

const withArt = {
  id: 'al-1',
  name: 'Album',
  blurHash: 'LEHV6nWB2yk8pyo0adR*.7kCMdnj',
}

describe('Artwork', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // jsdom has no 2D context; stub it so BlurHashCanvas bails cleanly without console noise
    HTMLCanvasElement.prototype.getContext = vi.fn(() => null)
  })

  it('renders nothing without a record', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: false })
    const { container } = render(<Artwork record={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('shows the blurhash and no <img> while loading', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(<Artwork record={withArt} title="Album" />)
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(container.querySelector('img')).toBeNull()
  })

  it('shows neither a broken <img> nor a canvas while loading a record with no blurhash', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(<Artwork record={{ id: 'al-2', name: 'X' }} />)
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('canvas')).toBeNull()
  })

  // The placeholder has to land exactly where the image will, or it jumps when the image swaps in.
  it('shapes the blurhash like the artwork and fits it like the image', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const nonSquare = { ...withArt, imageWidth: 1200, imageHeight: 800 }
    const { container } = render(
      <Artwork record={nonSquare} fit="contain" title="Album" />,
    )
    const canvas = container.querySelector('canvas')
    expect(canvas.width).toBe(32)
    expect(canvas.height).toBe(21)
    expect(canvas.style.objectFit).toBe('contain')
  })

  // A square request is padded, not cropped, so the artwork still sits letterboxed inside the
  // square the server returns and the placeholder has to letterbox with it.
  it('letterboxes the blurhash when the server pads a non-square image to a square', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const nonSquare = { ...withArt, imageWidth: 1200, imageHeight: 800 }
    const { container } = render(<Artwork record={nonSquare} square />)
    const canvas = container.querySelector('canvas')
    expect(canvas.width).toBe(32)
    expect(canvas.height).toBe(21)
    expect(canvas.style.objectFit).toBe('contain')
  })

  // The padded square the server returns is aspect-fit, so cropping it would disagree with the
  // placeholder. Both renderers have to read `square` the same way.
  it('fits the image itself with contain when the server padded to a square', () => {
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    const { container } = render(
      <Artwork record={withArt} square fit="cover" title="Album" />,
    )
    expect(container.querySelector('img').style.objectFit).toBe('contain')
  })

  it('fills the box for square artwork, the overwhelmingly common case', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const sq = { ...withArt, imageWidth: 600, imageHeight: 600 }
    const { container } = render(<Artwork record={sq} square />)
    const canvas = container.querySelector('canvas')
    expect(canvas.width).toBe(32)
    expect(canvas.height).toBe(32)
  })

  it('falls back to a square blurhash when the record has no dimensions', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(<Artwork record={withArt} />)
    const canvas = container.querySelector('canvas')
    expect(canvas.width).toBe(32)
    expect(canvas.height).toBe(32)
  })

  it('mounts the image only once its blob is ready', () => {
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    const { container } = render(<Artwork record={withArt} title="Album" />)
    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    expect(img.getAttribute('src')).toBe('blob:abc')
  })

  it('keeps the blurhash under the image until the fade ends', () => {
    vi.useFakeTimers()
    try {
      useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
      const { container, rerender } = render(<Artwork record={withArt} />)

      // Blob arrives: the image mounts transparent, with the blurhash still behind it.
      useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
      rerender(<Artwork record={withArt} />)
      const img = container.querySelector('img')
      expect(img).not.toBeNull()
      expect(container.querySelector('canvas')).not.toBeNull()

      // Decoding starts the cross-fade; the blurhash only goes away once it finishes. The clock
      // drives it, not transitionend, which never fires under prefers-reduced-motion.
      act(() => {
        fireEvent.load(img)
      })
      expect(container.querySelector('canvas')).not.toBeNull()
      act(() => {
        fireEvent.transitionEnd(img)
      })
      expect(container.querySelector('canvas')).not.toBeNull()
      act(() => {
        vi.advanceTimersByTime(500)
      })
      expect(container.querySelector('canvas')).toBeNull()
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not fade an image that was already cached on mount', () => {
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    const { container } = render(<Artwork record={withArt} />)
    // No placeholder to cross-fade from, so it paints at once.
    expect(container.querySelector('canvas')).toBeNull()
    expect(container.querySelector('img').className).toContain('imgInstant')
  })

  it('keeps the blurhash visible when the image never decodes', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container, rerender } = render(<Artwork record={withArt} />)
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    rerender(<Artwork record={withArt} />)

    expect(container.querySelector('canvas')).not.toBeNull()
  })

  it('fires onClick only when the image is loaded', () => {
    const onClick = vi.fn()
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container, rerender } = render(
      <Artwork record={withArt} onClick={onClick} />,
    )
    container.firstChild.click()
    expect(onClick).not.toHaveBeenCalled()

    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    rerender(<Artwork record={withArt} onClick={onClick} />)
    container.firstChild.click()
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})
