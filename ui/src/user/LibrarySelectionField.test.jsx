import * as React from 'react'
import { render, screen, cleanup } from '@testing-library/react'
import { LibrarySelectionField } from './LibrarySelectionField'
import { useInput, useTranslate } from 'react-admin'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { SelectLibraryInput } from '../common/SelectLibraryInput'

// Mock the react-admin hooks
vi.mock('react-admin', () => ({
  useInput: vi.fn(),
  useTranslate: vi.fn(),
}))

// Mock the SelectLibraryInput component
vi.mock('../common/SelectLibraryInput.jsx', () => ({
  SelectLibraryInput: vi.fn(() => <div data-testid="select-library-input" />),
}))

describe('<LibrarySelectionField />', () => {
  const defaultProps = {
    input: {
      name: 'libraryIds',
      value: [],
      onChange: vi.fn(),
    },
    meta: {
      touched: false,
      error: undefined,
    },
  }

  const mockTranslate = vi.fn((key) => key)

  beforeEach(() => {
    useInput.mockReturnValue(defaultProps)
    useTranslate.mockReturnValue(mockTranslate)
    SelectLibraryInput.mockClear()
  })

  afterEach(cleanup)

  it('should render field label from translations', () => {
    render(<LibrarySelectionField />)
    expect(screen.getByText('resources.user.fields.libraries')).not.toBeNull()
  })

  it('should render helper text from translations', () => {
    render(<LibrarySelectionField />)
    expect(
      screen.getByText('resources.user.helperTexts.libraries'),
    ).not.toBeNull()
  })

  it('should render SelectLibraryInput with correct props', () => {
    render(<LibrarySelectionField />)
    expect(screen.getByTestId('select-library-input')).not.toBeNull()
    expect(SelectLibraryInput).toHaveBeenCalledWith(
      expect.objectContaining({
        onChange: defaultProps.input.onChange,
        value: defaultProps.input.value,
      }),
      expect.anything(),
    )
  })

  it('should render error message when touched and has error', () => {
    useInput.mockReturnValue({
      ...defaultProps,
      meta: {
        touched: true,
        error: 'This field is required',
      },
    })

    render(<LibrarySelectionField />)
    expect(screen.getByText('This field is required')).not.toBeNull()
  })

  it('should not render error message when not touched', () => {
    useInput.mockReturnValue({
      ...defaultProps,
      meta: {
        touched: false,
        error: 'This field is required',
      },
    })

    render(<LibrarySelectionField />)
    expect(screen.queryByText('This field is required')).toBeNull()
  })

  it('should initialize with empty array when value is null', () => {
    useInput.mockReturnValue({
      ...defaultProps,
      input: {
        ...defaultProps.input,
        value: null,
      },
    })

    render(<LibrarySelectionField />)
    expect(SelectLibraryInput).toHaveBeenCalledWith(
      expect.objectContaining({
        value: [],
      }),
      expect.anything(),
    )
  })
})
