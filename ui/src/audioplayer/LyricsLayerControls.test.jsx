import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import React from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LyricsLayerControls from './LyricsLayerControls'

const renderControls = (props = {}) =>
  render(
    <LyricsLayerControls
      showTranslation={false}
      showPronunciation={false}
      translationEnabled={false}
      pronunciationEnabled={false}
      onToggleTranslation={() => {}}
      onTogglePronunciation={() => {}}
      {...props}
    />,
  )

describe('<LyricsLayerControls />', () => {
  afterEach(cleanup)

  it('toggles the experimental letter wave', () => {
    const onToggleLetterWave = vi.fn()
    renderControls({
      letterWave: true,
      letterWaveEnabled: true,
      onToggleLetterWave,
    })

    const button = screen.getByRole('button', {
      name: 'Turn off letter wave (experimental)',
    })
    expect(button).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(button)

    expect(onToggleLetterWave).toHaveBeenCalledTimes(1)
  })

  it('disables the letter wave when the lyrics have no word timing', () => {
    renderControls({ letterWave: false, letterWaveEnabled: false })

    expect(
      screen.getByRole('button', {
        name: 'Turn on letter wave (experimental)',
      }),
    ).toBeDisabled()
  })
})
