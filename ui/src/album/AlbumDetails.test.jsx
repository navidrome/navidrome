// ui/src/album/__tests__/AlbumDetails.test.jsx
import { describe, test, expect, beforeEach, afterEach } from 'vitest'
import { render } from '@testing-library/react'
import { RecordContextProvider } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { createTheme, ThemeProvider } from '@material-ui/core/styles'
import config from '../config'
import AlbumDetails, { Details } from './AlbumDetails'

vi.mock('../subsonic', () => ({
  default: {
    getAlbumInfo: () =>
      Promise.resolve({
        json: { 'subsonic-response': { status: 'ok', albumInfo: {} } },
      }),
    getCoverArtUrl: () => '',
  },
}))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useDataProvider: () => ({ getOne: vi.fn() }),
    useNotify: () => vi.fn(),
    useRefresh: () => vi.fn(),
  }
})

// Mock useMediaQuery
vi.mock('@material-ui/core', async () => {
  const actual = await import('@material-ui/core')
  return {
    ...actual,
    useMediaQuery: vi.fn(),
  }
})

// Mock formatFullDate to return deterministic results
vi.mock('../utils', async () => {
  const actual = await import('../utils')
  return {
    ...actual,
    formatFullDate: (date) => {
      if (!date) return ''
      // Use en-CA locale for consistent test results
      return new Date(date).toLocaleDateString('en-CA', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        timeZone: 'UTC',
      })
    },
  }
})

describe('Details component', () => {
  describe('Desktop view', () => {
    beforeEach(() => {
      // Set desktop view (isXsmall = false)
      vi.mocked(useMediaQuery).mockReturnValue(false)
    })

    test('renders correctly with just year range', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        year: 2020,
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with date', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        date: '2020-05-01',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with originalDate', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        originalDate: '2018-03-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with date and originalDate', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        date: '2020-05-01',
        originalDate: '2018-03-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with releaseDate', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        releaseDate: '2020-06-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with all date fields', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        date: '2020-05-01',
        originalDate: '2018-03-15',
        releaseDate: '2020-06-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })
  })

  describe('Mobile view', () => {
    beforeEach(() => {
      // Set mobile view (isXsmall = true)
      vi.mocked(useMediaQuery).mockReturnValue(true)
    })

    afterEach(() => {
      vi.clearAllMocks()
    })

    test('renders correctly with just year range', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        year: 2020,
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with date', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        date: '2020-05-01',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with originalDate', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        originalDate: '2018-03-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with date and originalDate', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        date: '2020-05-01',
        originalDate: '2018-03-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with releaseDate', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        releaseDate: '2020-06-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with all date fields', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        date: '2020-05-01',
        originalDate: '2018-03-15',
        releaseDate: '2020-06-15',
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with no date fields', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with year range (start and end years)', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        year: 2018,
        yearEnd: 2020,
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })

    test('renders correctly with originalYear range', () => {
      const record = {
        id: '123',
        name: 'Test Album',
        songCount: 12,
        duration: 3600,
        size: 102400,
        originalYear: 2015,
        originalYearEnd: 2016,
      }

      const { container } = render(
        <RecordContextProvider value={record}>
          <Details />
        </RecordContextProvider>,
      )

      expect(container).toMatchSnapshot()
    })
  })
})

describe('AlbumDetails cover animation', () => {
  const albumRecord = {
    id: '123',
    name: 'Test Album',
    songCount: 12,
    duration: 3600,
    size: 102400,
  }
  const originalEnableCoverAnimation = config.enableCoverAnimation

  beforeEach(() => {
    vi.mocked(useMediaQuery).mockReturnValue(false)
  })

  afterEach(() => {
    config.enableCoverAnimation = originalEnableCoverAnimation
  })

  const renderAlbum = () =>
    render(
      <ThemeProvider theme={createTheme()}>
        <RecordContextProvider value={albumRecord}>
          <AlbumDetails />
        </RecordContextProvider>
      </ThemeProvider>,
    )

  test('applies noCoverAnimation when enableCoverAnimation is false', () => {
    config.enableCoverAnimation = false
    const { container } = renderAlbum()
    const cover = container.querySelector('[class*="coverParent"]')

    expect(cover).not.toBeNull()
    expect(cover.className).toMatch(/noCoverAnimation/)
  })

  test('omits noCoverAnimation when enableCoverAnimation is true', () => {
    config.enableCoverAnimation = true
    const { container } = renderAlbum()
    const cover = container.querySelector('[class*="coverParent"]')

    expect(cover).not.toBeNull()
    expect(cover.className).not.toMatch(/noCoverAnimation/)
  })
})
