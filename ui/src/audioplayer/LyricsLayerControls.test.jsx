import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import React from 'react'
import { afterEach, describe, expect, it } from 'vitest'
import LyricsLayerControls from './LyricsLayerControls'

const labels = {
  viewSource: 'View lyrics source',
  sourceTitle: 'Lyrics source',
  embeddedSource: 'Embedded lyrics',
  sidecarSource: 'Sidecar file',
  pluginSource: 'Lyrics plugin',
  provider: 'Provider',
  format: 'Format',
}

const renderControls = (source) =>
  render(
    <LyricsLayerControls
      showTranslation={false}
      showPronunciation={false}
      translationEnabled={false}
      pronunciationEnabled={false}
      onToggleTranslation={() => {}}
      onTogglePronunciation={() => {}}
      labels={labels}
      source={source}
    />,
  )

describe('<LyricsLayerControls /> source details', () => {
  afterEach(cleanup)

  it('omits the source control when provenance is unavailable', () => {
    renderControls(null)

    expect(
      screen.queryByRole('button', { name: 'View lyrics source' }),
    ).not.toBeInTheDocument()
  })

  it.each([
    [{ type: 'embedded' }, 'Embedded lyrics', null],
    [{ type: 'sidecar', format: 'lrc' }, 'Sidecar file', 'LRC'],
  ])('shows built-in source details', (source, name, format) => {
    renderControls(source)

    fireEvent.click(screen.getByRole('button', { name: 'View lyrics source' }))

    expect(screen.getByText(name)).toBeVisible()
    if (format) expect(screen.getByText(format)).toBeVisible()
  })

  it('shows plugin, provider, and format without exposing extra metadata', async () => {
    renderControls({
      type: 'plugin',
      name: 'Better Lyrics',
      provider: 'unison',
      format: 'ttml',
      path: '/music/private/song.ttml',
    })

    const button = screen.getByRole('button', { name: 'View lyrics source' })
    fireEvent.click(button)

    expect(button).toHaveAttribute('aria-expanded', 'true')
    const dialog = screen.getByRole('dialog', { name: 'Lyrics source' })
    expect(dialog).toBeVisible()
    expect(screen.getByText('Better Lyrics')).toBeVisible()
    expect(screen.getByText('Unison')).toBeVisible()
    expect(screen.getByText('TTML')).toBeVisible()
    expect(
      screen.queryByText('/music/private/song.ttml'),
    ).not.toBeInTheDocument()

    fireEvent.keyDown(dialog.closest('[role="presentation"]'), {
      key: 'Escape',
      code: 'Escape',
    })
    await waitFor(() =>
      expect(button).toHaveAttribute('aria-expanded', 'false'),
    )
  })
})
