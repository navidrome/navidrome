import * as React from 'react'
import { TestContext } from 'ra-test'
import { render, screen, cleanup } from '@testing-library/react'
import { describe, afterEach, it, expect, vi } from 'vitest'
import DownloadMenuDialog from './DownloadMenuDialog'
import { DOWNLOAD_MENU_ALBUM, DOWNLOAD_MENU_ARTIST } from '../actions'

vi.mock('./useTranscodingOptions', () => ({
  useTranscodingOptions: () => ({
    TranscodingOptionsInput: () => null,
    format: '',
    maxBitRate: 0,
    originalFormat: true,
  }),
}))

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useTranslate: () => (key, opts) =>
      opts?.size ? `${key}:${opts.name}:${opts.size}` : key,
  }
})

const renderDialog = (record, recordType) =>
  render(
    <TestContext
      initialState={{ downloadMenuDialog: { open: true, record, recordType } }}
    >
      <DownloadMenuDialog />
    </TestContext>,
  )

describe('DownloadMenuDialog', () => {
  afterEach(cleanup)

  it('shows the album-artist size (not the total) for an artist download', () => {
    renderDialog(
      {
        id: 'ar1',
        name: 'Artist',
        size: 999999999,
        stats: { albumartist: { size: 1024 } },
      },
      DOWNLOAD_MENU_ARTIST,
    )
    expect(screen.getByText(/:Artist:1 KB$/)).toBeInTheDocument()
  })

  it('shows the total size for an album download', () => {
    renderDialog(
      { id: 'al1', name: 'Album', size: 1024 * 1024 },
      DOWNLOAD_MENU_ALBUM,
    )
    expect(screen.getByText(/:Album:1 MB$/)).toBeInTheDocument()
  })
})
