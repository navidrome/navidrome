import { render, screen, fireEvent, act } from '@testing-library/react'
import VolumeControl, { VOLUME_PROBE_MS } from './VolumeControl'

describe('VolumeControl', () => {
  let onVolumeChange
  let onMutedChange

  const renderControl = () =>
    render(
      <ul>
        <VolumeControl
          onVolumeChange={onVolumeChange}
          onMutedChange={onMutedChange}
        />
      </ul>,
    )

  beforeEach(() => {
    onVolumeChange = vi.fn()
    onMutedChange = vi.fn()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('starts at full volume', () => {
    renderControl()

    expect(screen.getByRole('slider')).toHaveAttribute('aria-valuenow', '1')
    expect(screen.getByText('100%')).toBeInTheDocument()
  })

  it('sends slider values to the player', () => {
    renderControl()
    const slider = screen.getByRole('slider')

    fireEvent.keyDown(slider, { key: 'PageDown' })
    fireEvent.keyDown(slider, { key: 'Home' })
    fireEvent.keyDown(slider, { key: 'End' })

    expect(onVolumeChange.mock.calls).toEqual([[0.9], [0], [1]])
  })

  it('mutes without losing the volume', () => {
    renderControl()
    fireEvent.keyDown(screen.getByRole('slider'), { key: 'PageDown' })

    fireEvent.click(screen.getByRole('button', { name: 'Mute' }))
    expect(onMutedChange).toHaveBeenLastCalledWith(true)
    expect(screen.getByRole('slider')).toHaveAttribute('aria-valuenow', '0')

    fireEvent.click(screen.getByRole('button', { name: 'Unmute' }))
    expect(onMutedChange).toHaveBeenLastCalledWith(false)
    expect(screen.getByRole('slider')).toHaveAttribute('aria-valuenow', '0.9')
    expect(onVolumeChange).toHaveBeenCalledTimes(1)
  })

  it('unmutes when the slider moves', () => {
    renderControl()
    fireEvent.click(screen.getByRole('button', { name: 'Mute' }))

    fireEvent.keyDown(screen.getByRole('slider'), { key: 'End' })

    expect(onMutedChange).toHaveBeenLastCalledWith(false)
    expect(onVolumeChange).toHaveBeenLastCalledWith(1)
    expect(screen.getByRole('button', { name: 'Mute' })).toBeInTheDocument()
  })

  it('keeps Space on the mute button away from the player play/pause key', () => {
    const { container } = renderControl()
    const playerKeyDown = vi.fn()
    container.addEventListener('keydown', playerKeyDown)

    fireEvent.keyDown(screen.getByRole('button', { name: 'Mute' }), {
      key: ' ',
      keyCode: 32,
    })

    expect(playerKeyDown).not.toHaveBeenCalled()
  })

  it('keeps the slider where the browser keeps the volume', () => {
    vi.useFakeTimers()
    renderControl()

    act(() => vi.advanceTimersByTime(VOLUME_PROBE_MS))

    expect(screen.getByRole('slider')).toBeInTheDocument()
  })

  it('shows only the mute button where the browser reverts the volume', () => {
    vi.useFakeTimers()
    let volume = 1
    vi.spyOn(HTMLMediaElement.prototype, 'volume', 'get').mockImplementation(
      () => volume,
    )
    vi.spyOn(HTMLMediaElement.prototype, 'volume', 'set').mockImplementation(
      (v) => {
        volume = v
        setTimeout(() => (volume = 1))
      },
    )
    renderControl()

    act(() => vi.advanceTimersByTime(VOLUME_PROBE_MS))

    expect(screen.queryByRole('slider')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Mute' }))
    expect(onMutedChange).toHaveBeenLastCalledWith(true)
  })
})
