import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { Form } from 'react-final-form'
import { TestContext } from 'ra-test'
import PIDAlbumInput from './PIDAlbumInput'

const renderWithForm = (initialValues, onSubmit = () => {}) =>
  render(
    <TestContext>
      <Form
        onSubmit={onSubmit}
        initialValues={initialValues}
        render={({ handleSubmit }) => (
          <form onSubmit={handleSubmit}>
            <PIDAlbumInput />
            <button type="submit">save</button>
          </form>
        )}
      />
    </TestContext>,
  )

describe('PIDAlbumInput', () => {
  it('hides the custom text box when the value is empty', () => {
    renderWithForm({ pidAlbum: '' })
    expect(screen.queryByTestId('pidAlbum-custom-input')).toBeNull()
  })

  it('hides the custom text box when the value is folder', () => {
    renderWithForm({ pidAlbum: 'folder' })
    expect(screen.queryByTestId('pidAlbum-custom-input')).toBeNull()
  })

  it('shows the custom text box for an unrecognized value', () => {
    renderWithForm({ pidAlbum: 'musicbrainz_albumid' })
    expect(screen.getByTestId('pidAlbum-custom-input')).toBeInTheDocument()
    expect(screen.getByDisplayValue('musicbrainz_albumid')).toBeInTheDocument()
  })

  it('blocks submit when the custom value is emptied', () => {
    const onSubmit = vi.fn()
    renderWithForm({ pidAlbum: 'musicbrainz_albumid' }, onSubmit)

    const input = screen
      .getByTestId('pidAlbum-custom-input')
      .querySelector('input')
    fireEvent.change(input, { target: { value: '' } })
    fireEvent.click(screen.getByText('save'))

    expect(onSubmit).not.toHaveBeenCalled()
    expect(
      screen.getByText('resources.library.validation.pidAlbumCustomRequired'),
    ).toBeInTheDocument()
  })
})
