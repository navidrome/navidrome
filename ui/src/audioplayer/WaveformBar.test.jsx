import React from 'react'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import WaveformBar from './WaveformBar'
import { loadWaveform } from './waveform'

vi.mock('./waveform', () => ({
  loadWaveform: vi.fn(),
  drawWaveform: vi.fn(),
}))

const renderProgressBar = () => {
  const root = document.createElement('div')
  root.className = 'react-jinke-music-player-main'
  root.innerHTML =
    '<div class="music-player-panel"><div class="progress-bar"><div class="rc-slider-track"></div></div></div>'
  document.body.appendChild(root)
  return root.querySelector('.progress-bar')
}

const renderMobileProgressBar = () => {
  const root = document.createElement('div')
  root.className = 'react-jinke-music-player-mobile'
  root.innerHTML =
    '<div class="react-jinke-music-player-mobile-progress"><div class="react-jinke-music-player-mobile-progress-bar"><div class="rc-slider-track"></div></div></div>'
  document.body.appendChild(root)
  return root
}

describe('<WaveformBar />', () => {
  afterEach(() => {
    cleanup()
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('draws the waveform inside the player progress bar', async () => {
    const progressBar = renderProgressBar()
    loadWaveform.mockResolvedValue([1, 2, 3])
    const onAvailable = vi.fn()

    render(<WaveformBar trackId="song-1" onAvailable={onAvailable} />)

    await waitFor(() => expect(onAvailable).toHaveBeenLastCalledWith(true))
    expect(progressBar.contains(screen.getByTestId('waveform'))).toBe(true)
    expect(loadWaveform).toHaveBeenCalledWith('song-1', expect.any(AbortSignal))
  })

  it('aborts the previous request when the track changes', async () => {
    renderProgressBar()
    loadWaveform.mockReturnValue(new Promise(() => {}))
    const onAvailable = vi.fn()

    const { rerender } = render(
      <WaveformBar trackId="song-1" onAvailable={onAvailable} />,
    )
    const firstSignal = loadWaveform.mock.calls[0][1]
    expect(firstSignal.aborted).toBe(false)

    rerender(<WaveformBar trackId="song-2" onAvailable={onAvailable} />)

    expect(firstSignal.aborted).toBe(true)
    expect(loadWaveform).toHaveBeenLastCalledWith(
      'song-2',
      expect.any(AbortSignal),
    )
    expect(loadWaveform.mock.calls[1][1].aborted).toBe(false)
  })

  it('renders nothing when the waveform is unavailable', async () => {
    renderProgressBar()
    loadWaveform.mockRejectedValue(new Error('disabled'))
    const onAvailable = vi.fn()

    render(<WaveformBar trackId="song-1" onAvailable={onAvailable} />)

    await waitFor(() => expect(loadWaveform).toHaveBeenCalled())
    expect(screen.queryByTestId('waveform')).toBeNull()
    expect(onAvailable).not.toHaveBeenCalledWith(true)
  })

  it('reports the waveform as gone when unmounted', async () => {
    renderProgressBar()
    loadWaveform.mockResolvedValue([1, 2, 3])
    const onAvailable = vi.fn()

    const { unmount } = render(
      <WaveformBar trackId="song-1" onAvailable={onAvailable} />,
    )
    await waitFor(() => expect(onAvailable).toHaveBeenLastCalledWith(true))
    unmount()
    expect(onAvailable).toHaveBeenLastCalledWith(false)
  })

  it('draws the waveform in the mobile player when it opens and drops it when it closes', async () => {
    loadWaveform.mockResolvedValue([1, 2, 3])
    const onAvailable = vi.fn()

    render(<WaveformBar trackId="song-1" onAvailable={onAvailable} />)
    await waitFor(() => expect(loadWaveform).toHaveBeenCalled())
    expect(screen.queryByTestId('waveform')).toBeNull()

    const mobile = renderMobileProgressBar()
    await waitFor(() => expect(onAvailable).toHaveBeenLastCalledWith(true))
    const progressBar = mobile.querySelector(
      '.react-jinke-music-player-mobile-progress-bar',
    )
    expect(progressBar.contains(screen.getByTestId('waveform'))).toBe(true)

    mobile.remove()
    await waitFor(() => expect(onAvailable).toHaveBeenLastCalledWith(false))
  })
})
