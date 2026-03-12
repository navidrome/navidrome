/* eslint-disable no-import-assign */
import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock ReactJkMusicPlayer - capture the customDownloader prop so tests can invoke it
let capturedCustomDownloader = null
vi.mock('navidrome-music-player', () => ({
  default: vi.fn(({ customDownloader, className }) => {
    capturedCustomDownloader = customDownloader
    return <div data-testid="music-player" className={className} />
  }),
}))

vi.mock('../config', () => ({
  default: {
    defaultLanguage: '',
    enableDownloads: true,
    publicBaseUrl: '/share',
    baseUrl: 'https://example.com',
  },
}))

vi.mock('../utils', async (importOriginal) => {
  const orig = await importOriginal()
  return {
    baseUrl: orig.baseUrl,
    shareStreamUrl: (id) => `/share/s/${id}`,
    shareCoverUrl: (id) => `/share/img/${id}`,
    shareDownloadUrl: (id) => `/share/d/${id}`,
    toDownloadUrl: (src) => `${src}?download=true`,
  }
})

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useTranslate: () => (key) => key,
  }
})

vi.mock('../dialogs/DialogTitle', () => ({
  DialogTitle: ({ children, onClose }) => (
    <div data-testid="dialog-title">
      {children}
      <button aria-label="close" onClick={onClose}>
        ×
      </button>
    </div>
  ),
}))

vi.mock('../dataProvider', async (importOriginal) => ({
  ...(await importOriginal()),
  httpClient: vi.fn(),
}))

// Spy on document.createElement to intercept anchor clicks
let createdLinks = []
const originalCreateElement = document.createElement.bind(document)

import { TestContext } from 'ra-test'
import SharePlayer from './SharePlayer'
import * as configModule from '../config'

describe('SharePlayer', () => {
  beforeEach(async () => {
    capturedCustomDownloader = null
    createdLinks = []
    configModule.shareInfo = {
      id: 'share-1',
      description: 'My Playlist',
      downloadable: true,
      tracks: [
        {
          id: 'track-1',
          title: 'Song One',
          artist: 'Artist One',
          duration: 180,
        },
        {
          id: 'track-2',
          title: 'Song Two',
          artist: 'Artist Two',
          duration: 240,
        },
      ],
    }

    vi.spyOn(document, 'createElement').mockImplementation((tag) => {
      const el = originalCreateElement(tag)
      if (tag === 'a') {
        vi.spyOn(el, 'click').mockImplementation(() => {})
        createdLinks.push(el)
      }
      return el
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    configModule.shareInfo = null
  })

  it('renders the music player', () => {
    render(
      <TestContext>
        <SharePlayer />
      </TestContext>,
    )
    expect(screen.getByTestId('music-player')).toBeInTheDocument()
  })

  it('does not show the download dialog initially', () => {
    render(
      <TestContext>
        <SharePlayer />
      </TestContext>,
    )
    // Dialog is closed: the MUI Dialog root should not be present or should be hidden
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  describe('customDownloader with multiple tracks', () => {
    it('opens the download dialog when customDownloader is called', () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })

      expect(
        screen.getByText('resources.share.actions.download.title'),
      ).toBeInTheDocument()
    })

    it('shows "Current Track" and "All Tracks" buttons in the dialog', () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })

      expect(
        screen.getByText('resources.share.actions.download.currentTrack'),
      ).toBeInTheDocument()
      expect(
        screen.getByText('resources.share.actions.download.allTracks'),
      ).toBeInTheDocument()
    })

    it('downloads the current track when "Current Track" is clicked', () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })

      fireEvent.click(
        screen.getByText('resources.share.actions.download.currentTrack'),
      )

      expect(createdLinks).toHaveLength(1)
      expect(createdLinks[0].href).toContain('download=true')
      expect(createdLinks[0].click).toHaveBeenCalled()
      expect(createdLinks[0]).not.toBeInTheDocument() // Link should be removed after click
    })

    it('closes the dialog after clicking "Current Track"', async () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })
      fireEvent.click(
        screen.getByText('resources.share.actions.download.currentTrack'),
      )

      await waitFor(() =>
        expect(
          screen.queryByText('resources.share.actions.download.title'),
        ).not.toBeInTheDocument(),
      )
    })

    it('downloads all tracks as zip when "All Tracks" is clicked', () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })

      fireEvent.click(
        screen.getByText('resources.share.actions.download.allTracks'),
      )

      expect(createdLinks).toHaveLength(1)
      expect(createdLinks[0].href).toContain('/share/d/share-1')
      expect(createdLinks[0].download).toBe('My Playlist.zip')
      expect(createdLinks[0].click).toHaveBeenCalled()
      expect(createdLinks[0]).not.toBeInTheDocument() 
    })

    it('closes the dialog after clicking "All Tracks"', async () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })
      fireEvent.click(
        screen.getByText('resources.share.actions.download.allTracks'),
      )

      await waitFor(() =>
        expect(
          screen.queryByText('resources.share.actions.download.title'),
        ).not.toBeInTheDocument(),
      )
    })

    it('closes the dialog when the close button is clicked', async () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })
      expect(
        screen.getByText('resources.share.actions.download.title'),
      ).toBeInTheDocument()

      fireEvent.click(screen.getByLabelText('close'))

      await waitFor(() =>
        expect(
          screen.queryByText('resources.share.actions.download.title'),
        ).not.toBeInTheDocument(),
      )
    })
  })

  describe('customDownloader with a single track', () => {
    beforeEach(() => {
      configModule.shareInfo = {
        id: 'share-2',
        description: 'Single Track',
        downloadable: true,
        tracks: [
          {
            id: 'track-1',
            title: 'Only Song',
            artist: 'Solo Artist',
            duration: 200,
          },
        ],
      }
    })

    it('downloads the track directly without opening a dialog', () => {
      render(
        <TestContext>
          <SharePlayer />
        </TestContext>,
      )

      capturedCustomDownloader({ src: '/share/s/track-1' })

      // Dialog title should NOT appear — direct download, no dialog
      expect(
        screen.queryByText('resources.share.actions.download.title'),
      ).not.toBeInTheDocument()
      // A link should have been clicked directly
      expect(createdLinks).toHaveLength(1)
      expect(createdLinks[0].href).toContain('download=true')
      expect(createdLinks[0].click).toHaveBeenCalled()
      expect(createdLinks[0]).not.toBeInTheDocument() 
    })
  })
})
