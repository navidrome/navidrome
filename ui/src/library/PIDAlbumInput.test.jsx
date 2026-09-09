import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Form } from 'react-final-form'
import { TestContext } from 'ra-test'
import PIDAlbumInput from './PIDAlbumInput'

const renderWithForm = (initialValues) =>
  render(
    <TestContext>
      <Form
        onSubmit={() => {}}
        initialValues={initialValues}
        render={() => <PIDAlbumInput />}
      />
    </TestContext>,
  )

describe('PIDAlbumInput', () => {
  it('hides the custom text box when the value is empty', () => {
    renderWithForm({ pidAlbum: '' })
    expect(screen.queryByLabelText(/Album Grouping \(custom\)/i)).toBeNull()
  })

  it('hides the custom text box when the value is folder', () => {
    renderWithForm({ pidAlbum: 'folder' })
    expect(screen.queryByLabelText(/Album Grouping \(custom\)/i)).toBeNull()
  })

  it('shows the custom text box for an unrecognized value', () => {
    renderWithForm({ pidAlbum: 'musicbrainz_albumid' })
    expect(screen.getByDisplayValue('musicbrainz_albumid')).toBeInTheDocument()
  })
})
