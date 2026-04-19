import React from 'react'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import MobileKaraokeLyricsPortal, {
  MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS,
} from './MobileKaraokeLyricsPortal'

const HOST_CLASS = 'react-jinke-music-player-mobile-cover'

describe('<MobileKaraokeLyricsPortal />', () => {
  afterEach(() => {
    cleanup()
    document.body.innerHTML = ''
  })

  it('renders lyrics into the mobile cover host and toggles the active class', () => {
    const host = document.createElement('div')
    host.className = HOST_CLASS
    document.body.appendChild(host)

    const { rerender } = render(
      <MobileKaraokeLyricsPortal active={true}>
        <div data-testid="mobile-inline-lyrics">Lyrics</div>
      </MobileKaraokeLyricsPortal>,
    )

    expect(host).toContainElement(screen.getByTestId('mobile-inline-lyrics'))
    expect(host).toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)

    rerender(
      <MobileKaraokeLyricsPortal active={false}>
        <div data-testid="mobile-inline-lyrics">Lyrics</div>
      </MobileKaraokeLyricsPortal>,
    )

    expect(screen.queryByTestId('mobile-inline-lyrics')).not.toBeInTheDocument()
    expect(host).not.toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
  })

  it('attaches when the mobile cover host appears after mount', async () => {
    render(
      <MobileKaraokeLyricsPortal active={true}>
        <div data-testid="mobile-inline-lyrics">Lyrics</div>
      </MobileKaraokeLyricsPortal>,
    )

    const host = document.createElement('div')
    host.className = HOST_CLASS
    document.body.appendChild(host)

    await waitFor(() =>
      expect(host).toContainElement(screen.getByTestId('mobile-inline-lyrics')),
    )
    expect(host).toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
  })
})
