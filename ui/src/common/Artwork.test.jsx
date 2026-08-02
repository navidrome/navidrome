import { render, fireEvent, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('./useImageUrl', () => ({ useImageUrl: vi.fn() }))
vi.mock('../subsonic', () => ({
  default: {
    // Mirrors the real hash suffix, so a refreshed record yields a different URL.
    getCoverArtUrl: (record) =>
      '/rest/getCoverArt?id=al-1' +
      (record.imageHash ? `_${record.imageHash}` : ''),
  },
}))
vi.mock('../config', () => ({ default: { uiCoverArtSize: 300 } }))

import { useImageUrl } from './useImageUrl'
import { Artwork } from './Artwork'

const withArt = {
  id: 'al-1',
  name: 'Album',
  thumbHash: 'H/gNBxpwh4dwd3eIiHd3iHeHeJ+dcH8I',
}

describe('Artwork', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // jsdom has no 2D context; stub it so ThumbHashCanvas bails cleanly without console noise
    HTMLCanvasElement.prototype.getContext = vi.fn(() => null)
  })

  it('renders nothing without a record', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: false })
    const { container } = render(<Artwork record={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('shows the placeholder and no <img> while loading', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(<Artwork record={withArt} title="Album" />)
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(container.querySelector('img')).toBeNull()
  })

  it('shows neither a broken <img> nor a canvas while loading a record with no thumbhash', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(<Artwork record={{ id: 'al-2', name: 'X' }} />)
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('canvas')).toBeNull()
  })

  // The placeholder has to land exactly where the image will, or it jumps when the image swaps in.
  it('shapes the placeholder like the artwork and fits it like the image', () => {
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

  // A square request is padded, not cropped, so the placeholder has to letterbox with it.
  it('letterboxes the placeholder when the server pads a non-square image to a square', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const nonSquare = { ...withArt, imageWidth: 1200, imageHeight: 800 }
    const { container } = render(<Artwork record={nonSquare} square />)
    const canvas = container.querySelector('canvas')
    expect(canvas.width).toBe(32)
    expect(canvas.height).toBe(21)
    expect(canvas.style.objectFit).toBe('contain')
  })

  // The padded square is aspect-fit, so `square` overrides fit="cover" as it does for the canvas.
  it('fits the image itself with contain when the server padded to a square', () => {
    useImageUrl.mockReturnValue({
      imgUrl: 'blob:abc',
      loading: false,
      fromCache: true,
    })
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

  // Without dimensions the placeholder falls back to the aspect the hash itself carries.
  it('falls back to the hash own aspect when the record has no dimensions', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(<Artwork record={withArt} />)
    const canvas = container.querySelector('canvas')
    expect(canvas.width).toBe(32)
    expect(canvas.height).toBe(32)
  })

  it('mounts the image only once its blob is ready', () => {
    useImageUrl.mockReturnValue({
      imgUrl: 'blob:abc',
      loading: false,
      fromCache: true,
    })
    const { container } = render(<Artwork record={withArt} title="Album" />)
    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    expect(img.getAttribute('src')).toBe('blob:abc')
  })

  it('keeps the placeholder under the image until the fade ends', () => {
    vi.useFakeTimers()
    try {
      useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
      const { container, rerender } = render(<Artwork record={withArt} />)

      useImageUrl.mockReturnValue({
        imgUrl: 'blob:abc',
        loading: false,
        fromCache: false,
      })
      rerender(<Artwork record={withArt} />)
      const img = container.querySelector('img')
      expect(img).not.toBeNull()
      expect(container.querySelector('canvas')).not.toBeNull()

      // The clock ends the fade, not transitionend, which never fires under reduced-motion.
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
    useImageUrl.mockReturnValue({
      imgUrl: 'blob:abc',
      loading: false,
      fromCache: true,
    })
    const { container } = render(<Artwork record={withArt} />)
    expect(container.querySelector('canvas')).toBeNull()
    expect(container.querySelector('img').className).toContain('imgInstant')
  })

  it('keeps the placeholder visible when the image never decodes', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container, rerender } = render(<Artwork record={withArt} />)
    useImageUrl.mockReturnValue({
      imgUrl: 'blob:abc',
      loading: false,
      fromCache: false,
    })
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

    useImageUrl.mockReturnValue({
      imgUrl: 'blob:abc',
      loading: false,
      fromCache: false,
    })
    rerender(<Artwork record={withArt} onClick={onClick} />)
    container.firstChild.click()
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('falls back to the placeholder when a refresh swaps in an uncached hash', () => {
    useImageUrl.mockReturnValue({
      imgUrl: 'blob:old',
      loading: false,
      fromCache: true,
    })
    const cached = { ...withArt, imageHash: 'aaaa' }
    const { container, rerender } = render(<Artwork record={cached} />)
    expect(container.querySelector('canvas')).toBeNull()

    useImageUrl.mockReturnValue({
      imgUrl: null,
      loading: true,
      fromCache: false,
    })
    rerender(<Artwork record={{ ...withArt, imageHash: 'bbbb' }} />)

    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('canvas')).not.toBeNull()
  })
})
