import { render, act } from '@testing-library/react'
import SharePlayer from './SharePlayer'

let playerProps

vi.mock('navidrome-music-player', () => ({
  default: (props) => {
    playerProps = props
    return <div data-testid="player" />
  },
}))

vi.mock('../config', () => ({
  default: { enableDownloads: true },
  shareInfo: {
    id: 'share-1',
    downloadable: true,
    tracks: [{ id: 't1', title: 'One', artist: 'A', duration: 100 }],
  },
}))

vi.mock('../utils', () => ({
  shareDownloadUrl: (id) => `/share/d/${id}`,
  shareStreamUrl: (id) => `/share/s/${id}`,
  shareCoverUrl: (id) => `/share/img/${id}`,
}))

describe('SharePlayer', () => {
  let clickSpy

  beforeEach(() => {
    vi.useFakeTimers()
    playerProps = null
    // Downloading for real would navigate the jsdom window.
    clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => {})
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('downloads via an anchor so the service worker does not intercept it', () => {
    render(<SharePlayer />)

    let anchor
    clickSpy.mockImplementation(function () {
      anchor = { href: this.href, download: this.download }
    })

    act(() => {
      playerProps.customDownloader()
    })

    expect(clickSpy).toHaveBeenCalledTimes(1)
    expect(anchor.href).toContain('/share/d/share-1')
    // Empty, so the server's Content-Disposition filename wins.
    expect(anchor.download).toBe('')
  })

  it('removes the anchor from the document after clicking', () => {
    render(<SharePlayer />)

    act(() => {
      playerProps.customDownloader()
    })

    expect(document.querySelectorAll('a[download]')).toHaveLength(0)
  })

  // The inert styling itself is driven by JSS function values, which jsdom does
  // not evaluate; it is verified in a browser. What is checked here is the
  // state machine feeding it -- that the component re-renders on download and
  // again when the window closes.
  it('re-renders when the feedback window opens and closes', () => {
    render(<SharePlayer />)
    const renders = []
    const track = () => renders.push(playerProps)

    track()
    act(() => {
      playerProps.customDownloader()
    })
    track()

    act(() => {
      vi.advanceTimersByTime(2000)
    })
    track()

    expect(renders[1]).not.toBe(renders[0])
    expect(renders[2]).not.toBe(renders[1])
  })

  it('restarts the feedback window on a repeat download', () => {
    render(<SharePlayer />)

    act(() => {
      playerProps.customDownloader()
    })
    act(() => {
      vi.advanceTimersByTime(1500)
    })
    act(() => {
      playerProps.customDownloader()
    })

    // Would have elapsed had the first timer not been replaced.
    act(() => {
      vi.advanceTimersByTime(1000)
    })
    const midway = playerProps

    act(() => {
      vi.advanceTimersByTime(1000)
    })

    expect(playerProps).not.toBe(midway)
  })

  it('does not update state after unmount', () => {
    const { unmount } = render(<SharePlayer />)
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    act(() => {
      playerProps.customDownloader()
    })
    unmount()
    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(errorSpy).not.toHaveBeenCalled()
  })
})
