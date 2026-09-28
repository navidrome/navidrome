import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  ReadOnlyDateField,
  ReadOnlyDurationField,
  ReadOnlyNumberField,
  ReadOnlySizeField,
  ReadOnlyTextField,
} from './ReadOnlyFields'

vi.mock('react-admin', async (importOriginal) => ({
  ...(await importOriginal()),
  useLocale: vi.fn(),
}))

describe('ReadOnlyFields', () => {
  const record = {
    id: '1',
    client: 'NavidromeUI',
    createdAt: '2026-09-17T14:30:00Z',
    lastVisitedAt: '0001-01-01T00:00:00Z',
    count: 1234567,
    zero: 0,
    size: 1536000,
    duration: 3725,
  }

  beforeEach(async () => {
    vi.clearAllMocks()
    vi.spyOn(navigator, 'languages', 'get').mockReturnValue([])
    const { useLocale } = await import('react-admin')
    vi.mocked(useLocale).mockReturnValue('de')
  })

  const renderField = (Field, props) =>
    render(<Field record={record} resource="player" {...props} />)

  describe('<ReadOnlyTextField>', () => {
    it('shows the record value with the translated field label', () => {
      renderField(ReadOnlyTextField, { source: 'client' })
      const input = screen.getByLabelText('resources.player.fields.client')
      expect(input).toHaveValue('NavidromeUI')
    })

    it('cannot be edited or reached with the Tab key', () => {
      renderField(ReadOnlyTextField, { source: 'client' })
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('readonly')
      expect(input).toHaveAttribute('tabindex', '-1')
    })

    it('shows an empty value when the record has no value', () => {
      renderField(ReadOnlyTextField, { source: 'userName' })
      expect(screen.getByRole('textbox')).toHaveValue('')
    })

    it('uses an explicit label when given', () => {
      renderField(ReadOnlyTextField, { source: 'client', label: 'Custom' })
      expect(screen.getByLabelText('Custom')).toBeInTheDocument()
    })

    it('applies a custom format', () => {
      renderField(ReadOnlyTextField, {
        source: 'client',
        format: (v) => v.toUpperCase(),
      })
      expect(screen.getByRole('textbox')).toHaveValue('NAVIDROMEUI')
    })

    it('exposes theme-overridable class names', () => {
      const { container } = renderField(ReadOnlyTextField, {
        source: 'client',
      })
      expect(
        container.querySelector('[class*="NDReadOnlyField-inputRoot"]'),
      ).toBeInTheDocument()
      expect(
        container.querySelector('[class*="NDReadOnlyField-notchedOutline"]'),
      ).toBeInTheDocument()
    })
  })

  describe('<ReadOnlyDateField>', () => {
    it('formats the date using the selected language', () => {
      renderField(ReadOnlyDateField, { source: 'createdAt' })
      expect(screen.getByRole('textbox').value).toMatch(/^17\.9\.2026, /)
    })

    it('shows an empty value when the date is not set', () => {
      renderField(ReadOnlyDateField, { source: 'lastVisitedAt' })
      expect(screen.getByRole('textbox')).toHaveValue('')
    })
  })

  describe('<ReadOnlyNumberField>', () => {
    it('formats the number using the selected language', () => {
      renderField(ReadOnlyNumberField, { source: 'count' })
      expect(screen.getByRole('textbox')).toHaveValue('1.234.567')
    })

    it('shows zero', () => {
      renderField(ReadOnlyNumberField, { source: 'zero' })
      expect(screen.getByRole('textbox')).toHaveValue('0')
    })
  })

  describe('<ReadOnlySizeField>', () => {
    it('formats bytes as a human-readable size', () => {
      renderField(ReadOnlySizeField, { source: 'size' })
      expect(screen.getByRole('textbox')).toHaveValue('1.46 MB')
    })
  })

  describe('<ReadOnlyDurationField>', () => {
    it('formats seconds as a human-readable duration', () => {
      renderField(ReadOnlyDurationField, { source: 'duration' })
      expect(screen.getByRole('textbox')).toHaveValue('1h 2m 5s')
    })
  })
})
