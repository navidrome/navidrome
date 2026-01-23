import { describe, test, expect, vi } from 'vitest'
import { render } from '@testing-library/react'
import { RecordContextProvider } from 'react-admin'
import { AlbumDatesField } from './AlbumDatesField'
import { formatRange } from '../common/index.js'

// Mock the formatRange function
vi.mock('../common/index.js', () => ({
  formatRange: vi.fn(),
}))

describe('AlbumDatesField', () => {
  test('renders nothing when yearRange is "0"', () => {
    const record = {
      maxYear: '0',
      releaseDate: '2020-01-01',
    }

    vi.mocked(formatRange).mockReturnValue('0')

    const { container } = render(
      <RecordContextProvider value={record}>
        <AlbumDatesField />
      </RecordContextProvider>,
    )

    expect(container.firstChild).toBeNull()
  })

  test('renders nothing when releaseYear is "0"', () => {
    const record = {
      maxYear: '2020',
      releaseDate: '0-01-01',
    }

    vi.mocked(formatRange).mockReturnValue('2020')

    const { container } = render(
      <RecordContextProvider value={record}>
        <AlbumDatesField />
      </RecordContextProvider>,
    )

    expect(container.firstChild).toBeNull()
  })

  test('renders only yearRange when releaseYear is undefined', () => {
    const record = {
      maxYear: '2020',
    }

    vi.mocked(formatRange).mockReturnValue('2020')

    const { container } = render(
      <RecordContextProvider value={record}>
        <AlbumDatesField />
      </RecordContextProvider>,
    )

    expect(container.textContent).toBe('2020')
  })

  test('renders both years when they are different', () => {
    const record = {
      maxYear: '2018',
      releaseDate: '2020-01-01',
    }

    vi.mocked(formatRange).mockReturnValue('2018')

    const { container } = render(
      <RecordContextProvider value={record}>
        <AlbumDatesField />
      </RecordContextProvider>,
    )

    expect(container.textContent).toBe('♫ 2018 · ○ 2020')
  })

  test('renders only yearRange when both years are the same', () => {
    const record = {
      maxYear: '2020',
      releaseDate: '2020-01-01',
    }

    vi.mocked(formatRange).mockReturnValue('2020')

    const { container } = render(
      <RecordContextProvider value={record}>
        <AlbumDatesField />
      </RecordContextProvider>,
    )

    expect(container.textContent).toBe('2020')
  })

  test('applies className when provided', () => {
    const record = {
      maxYear: '2020',
    }

    vi.mocked(formatRange).mockReturnValue('2020')

    const { container } = render(
      <RecordContextProvider value={record}>
        <AlbumDatesField className="test-class" />
      </RecordContextProvider>,
    )

    expect(container.firstChild).toHaveClass('test-class')
  })
})
