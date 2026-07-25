import { render, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('./useImageUrl', () => ({ useImageUrl: vi.fn() }))
vi.mock('../subsonic', () => ({
  default: { getCoverArtUrl: () => '/rest/getCoverArt?id=al-1' },
}))
vi.mock('../config', () => ({ default: { uiCoverArtSize: 300 } }))

import { useImageUrl } from './useImageUrl'
import { CoverImage } from './CoverImage'

const withArt = {
  id: 'al-1',
  name: 'Album',
  blurHash: 'LEHV6nWB2yk8pyo0adR*.7kCMdnj',
}

describe('CoverImage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // jsdom has no 2D context; stub it so BlurHashCanvas bails cleanly without console noise
    HTMLCanvasElement.prototype.getContext = vi.fn(() => null)
  })

  it('renders nothing without a record', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: false })
    const { container } = render(<CoverImage record={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('shows the blurhash and no <img> while loading', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(<CoverImage record={withArt} title="Album" />)
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(container.querySelector('img')).toBeNull()
  })

  it('shows neither a broken <img> nor a canvas while loading a record with no blurhash', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container } = render(
      <CoverImage record={{ id: 'al-2', name: 'X' }} />,
    )
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('canvas')).toBeNull()
  })

  it('mounts the image only once its blob is ready', () => {
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    const { container } = render(<CoverImage record={withArt} title="Album" />)
    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    expect(img.getAttribute('src')).toBe('blob:abc')
  })

  it('keeps the blurhash under the image until the fade ends', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container, rerender } = render(<CoverImage record={withArt} />)

    // Blob arrives: the image mounts transparent, with the blurhash still behind it.
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    rerender(<CoverImage record={withArt} />)
    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    expect(container.querySelector('canvas')).not.toBeNull()

    // Decoding starts the cross-fade; the blurhash only goes away once it finishes.
    fireEvent.load(img)
    expect(container.querySelector('canvas')).not.toBeNull()
    fireEvent.transitionEnd(img)
    expect(container.querySelector('canvas')).toBeNull()
  })

  it('does not fade an image that was already cached on mount', () => {
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    const { container } = render(<CoverImage record={withArt} />)
    // No placeholder to cross-fade from, so it paints at once.
    expect(container.querySelector('canvas')).toBeNull()
    expect(container.querySelector('img').className).toContain('imgInstant')
  })

  it('keeps the blurhash visible when the image never decodes', () => {
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container, rerender } = render(<CoverImage record={withArt} />)
    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    rerender(<CoverImage record={withArt} />)

    expect(container.querySelector('canvas')).not.toBeNull()
  })

  it('fires onClick only when the image is loaded', () => {
    const onClick = vi.fn()
    useImageUrl.mockReturnValue({ imgUrl: null, loading: true })
    const { container, rerender } = render(
      <CoverImage record={withArt} onClick={onClick} />,
    )
    container.firstChild.click()
    expect(onClick).not.toHaveBeenCalled()

    useImageUrl.mockReturnValue({ imgUrl: 'blob:abc', loading: false })
    rerender(<CoverImage record={withArt} onClick={onClick} />)
    container.firstChild.click()
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})
