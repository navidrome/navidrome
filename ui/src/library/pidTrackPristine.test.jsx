import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Form, FormSpy } from 'react-final-form'
import { TestContext } from 'ra-test'
import { TextInput } from 'react-admin'
import { parsePidField } from './pidUtils'

// Mirrors the pidTrack TextInput used in LibraryEdit.jsx / LibraryCreate.jsx.
const renderPidTrackField = (initialValues) => {
  const pristineHistory = []
  render(
    <TestContext>
      <Form
        onSubmit={() => {}}
        initialValues={initialValues}
        render={({ handleSubmit }) => (
          <form onSubmit={handleSubmit}>
            <TextInput source="pidTrack" parse={parsePidField} />
            <FormSpy
              subscription={{ pristine: true }}
              onChange={({ pristine }) => pristineHistory.push(pristine)}
            />
          </form>
        )}
      />
    </TestContext>,
  )
  return pristineHistory
}

describe('pidTrack field pristine behavior', () => {
  it('returns to pristine when the value is reverted back to the original empty value', async () => {
    const pristineHistory = renderPidTrackField({ pidTrack: '' })
    expect(pristineHistory[pristineHistory.length - 1]).toBe(true)

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: 'title' } })
    await waitFor(() =>
      expect(pristineHistory[pristineHistory.length - 1]).toBe(false),
    )

    fireEvent.change(input, { target: { value: '' } })
    await waitFor(() =>
      expect(pristineHistory[pristineHistory.length - 1]).toBe(true),
    )
  })
})
