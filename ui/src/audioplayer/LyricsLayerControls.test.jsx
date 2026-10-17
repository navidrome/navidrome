import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import React from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LyricsLayerControls from './LyricsLayerControls'

const labels = {
  viewSource: 'View lyrics source',
  sourceTitle: 'Lyrics source',
  autoSource: 'Auto (best match)',
  wordSynced: 'Word-synced',
  lineSynced: 'Line-synced',
  plainSynced: 'Plain text',
  embeddedSource: 'Embedded lyrics',
  sidecarSource: 'Sidecar file',
  pluginSource: 'Lyrics plugin',
  refreshingSources: 'Looking for more sources',
  offset: 'Lyrics timing offset',
  offsetCoarseEarlier: 'Shift lyrics 0.5 s earlier',
  offsetFineEarlier: 'Shift lyrics 0.05 s earlier',
  offsetCoarseLater: 'Shift lyrics 0.5 s later',
  offsetFineLater: 'Shift lyrics 0.05 s later',
  offsetReset: 'Reset timing offset',
  download: 'Download lyrics',
}

const variants = [
  {
    key: 'netease',
    source: {
      type: 'plugin',
      name: 'Better Lyrics',
      provider: 'NetEase',
      format: 'elrc',
    },
    sync: 'word',
  },
  {
    key: 'lrclib',
    source: {
      type: 'plugin',
      name: 'Better Lyrics',
      provider: 'LRCLib',
      format: 'lrc',
    },
    sync: 'line',
  },
]

const renderControls = (props = {}) =>
  render(
    <LyricsLayerControls
      showTranslation={false}
      showPronunciation={false}
      translationEnabled={false}
      pronunciationEnabled={false}
      onToggleTranslation={() => {}}
      onTogglePronunciation={() => {}}
      labels={labels}
      variants={variants}
      {...props}
    />,
  )

const openSourceMenu = () =>
  fireEvent.click(screen.getByRole('button', { name: labels.viewSource }))

describe('<LyricsLayerControls />', () => {
  afterEach(cleanup)

  it('omits the source and download controls when no variant has lyrics', () => {
    renderControls({ variants: [] })

    expect(
      screen.queryByRole('button', { name: labels.viewSource }),
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: labels.download })).toBeDisabled()
  })

  it('lists auto and every variant with badges and checked state', () => {
    renderControls({ selectedSourceKey: 'lrclib' })

    const button = screen.getByRole('button', { name: labels.viewSource })
    fireEvent.click(button)

    expect(button).toHaveAttribute('aria-expanded', 'true')
    expect(
      screen.getByRole('menuitemradio', { name: labels.autoSource }),
    ).toHaveAttribute('aria-checked', 'false')

    const netease = screen.getByRole('menuitemradio', {
      name: /NetEase/,
    })
    expect(netease).toHaveAttribute('aria-checked', 'false')
    expect(netease).toHaveTextContent('ELRC')
    expect(netease).toHaveTextContent('Word-synced')

    const lrclib = screen.getByRole('menuitemradio', { name: /LRCLib/ })
    expect(lrclib).toHaveAttribute('aria-checked', 'true')
    expect(lrclib).toHaveTextContent('LRC')
    expect(lrclib).toHaveTextContent('Line-synced')
  })

  it('names built-in sources and hides extra metadata', () => {
    renderControls({
      variants: [
        {
          key: 'embedded',
          source: { type: 'embedded', path: '/music/private/song.ttml' },
          sync: 'plain',
        },
      ],
    })

    openSourceMenu()

    expect(screen.getByText('Embedded lyrics')).toBeVisible()
    expect(screen.getByText('Plain text')).toBeVisible()
    expect(
      screen.queryByText('/music/private/song.ttml'),
    ).not.toBeInTheDocument()
  })

  it('calls onSelectSource with the variant key and null for auto', () => {
    const onSelectSource = vi.fn()
    renderControls({ onSelectSource })

    openSourceMenu()
    fireEvent.click(screen.getByRole('menuitemradio', { name: /LRCLib/ }))
    expect(onSelectSource).toHaveBeenLastCalledWith('lrclib')

    openSourceMenu()
    fireEvent.click(
      screen.getByRole('menuitemradio', { name: labels.autoSource }),
    )
    expect(onSelectSource).toHaveBeenLastCalledWith(null)
  })

  it('signals the source menu opening and closing', async () => {
    const onSourceMenuOpenChange = vi.fn()
    renderControls({ onSourceMenuOpenChange })

    openSourceMenu()
    expect(onSourceMenuOpenChange).toHaveBeenLastCalledWith(true)

    const menu = screen.getByRole('menu')
    fireEvent.keyDown(menu.closest('[role="presentation"]'), {
      key: 'Escape',
      code: 'Escape',
    })
    await waitFor(() =>
      expect(onSourceMenuOpenChange).toHaveBeenLastCalledWith(false),
    )
  })

  it('steps and clamps the offset and resets to zero', () => {
    const onOffsetChange = vi.fn()
    renderControls({ offsetMs: 1500, offsetEnabled: true, onOffsetChange })

    fireEvent.click(
      screen.getByRole('button', {
        name: `${labels.offset}: +1.50 s`,
      }),
    )

    fireEvent.click(
      screen.getByRole('button', { name: labels.offsetFineLater }),
    )
    expect(onOffsetChange).toHaveBeenLastCalledWith(1550)

    fireEvent.click(
      screen.getByRole('button', { name: labels.offsetCoarseEarlier }),
    )
    expect(onOffsetChange).toHaveBeenLastCalledWith(1000)

    fireEvent.click(screen.getByRole('button', { name: labels.offsetReset }))
    expect(onOffsetChange).toHaveBeenLastCalledWith(0)
  })

  it('clamps the offset at the limit', () => {
    const onOffsetChange = vi.fn()
    renderControls({ offsetMs: 5000, offsetEnabled: true, onOffsetChange })

    fireEvent.click(
      screen.getByRole('button', { name: `${labels.offset}: +5.00 s` }),
    )
    fireEvent.click(
      screen.getByRole('button', { name: labels.offsetCoarseLater }),
    )

    expect(onOffsetChange).toHaveBeenLastCalledWith(5000)
  })

  it('disables the offset control when the selected variant has no timing', () => {
    renderControls({ offsetEnabled: false })

    expect(
      screen.getByRole('button', {
        name: `${labels.offset}: 0.00 s`,
      }),
    ).toBeDisabled()
  })

  it('calls onDownload when the download button is pressed', () => {
    const onDownload = vi.fn()
    renderControls({ onDownload })

    fireEvent.click(screen.getByRole('button', { name: labels.download }))

    expect(onDownload).toHaveBeenCalledTimes(1)
  })
})
