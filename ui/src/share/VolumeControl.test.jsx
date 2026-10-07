import { render, screen, fireEvent, act } from '@testing-library/react'
import VolumeControl from './VolumeControl'

const renderControl = (audio) =>
  render(
    <ul>
      <VolumeControl audio={audio} />
    </ul>,
  )

describe('VolumeControl', () => {
  let audio

  beforeEach(() => {
    audio = document.createElement('audio')
  })

  it('reflects the current audio volume on the slider scale', () => {
    audio.volume = 0.25
    renderControl(audio)

    // The slider uses a square-root scale, like the player's own volume bar
    expect(screen.getByRole('slider')).toHaveAttribute('aria-valuenow', '0.5')
  })

  it('changes the audio volume using the square of the slider value', () => {
    renderControl(audio)
    const slider = screen.getByRole('slider')

    fireEvent.keyDown(slider, { key: 'PageDown' })

    // 1 -> 0.9 on the slider, 0.81 real volume
    expect(audio.volume).toBeCloseTo(0.81)

    fireEvent.keyDown(slider, { key: 'Home' })
    expect(audio.volume).toBe(0)
  })

  it('toggles mute and unmutes when the slider is moved', () => {
    renderControl(audio)

    fireEvent.click(screen.getByRole('button', { name: 'Mute' }))
    expect(audio.muted).toBe(true)
    // jsdom does not dispatch volumechange by itself
    act(() => {
      audio.dispatchEvent(new Event('volumechange'))
    })
    expect(screen.getByRole('button', { name: 'Unmute' })).toBeInTheDocument()

    fireEvent.keyDown(screen.getByRole('slider'), { key: 'End' })
    expect(audio.muted).toBe(false)
  })

  it('follows volume changes made elsewhere', () => {
    renderControl(audio)

    act(() => {
      audio.volume = 0.04
      audio.dispatchEvent(new Event('volumechange'))
    })

    expect(screen.getByRole('slider')).toHaveAttribute('aria-valuenow', '0.2')
  })

  it('is disabled until the audio element is available', () => {
    renderControl(null)

    expect(screen.getByRole('slider').closest('.MuiSlider-root')).toHaveClass(
      'Mui-disabled',
    )
  })
})
