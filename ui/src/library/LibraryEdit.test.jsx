import * as React from 'react'
import { TestContext } from 'ra-test'
import {
  FormWithRedirect,
  RecordContextProvider,
  SaveContextProvider,
} from 'react-admin'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { LibraryEditForm } from './LibraryEdit'
import config from '../config'

const record = {
  id: '2',
  name: 'Jazz',
  path: '/music/jazz',
  pidAlbum: '',
  pidTrack: '',
}

// Edit provides a save context in the app. SaveButton only reads these setters from it
const saveContext = {
  save: vi.fn(),
  setOnSuccess: vi.fn(),
  setOnFailure: vi.fn(),
  setTransform: vi.fn(),
}

const renderForm = (save) =>
  render(
    <TestContext>
      <SaveContextProvider value={saveContext}>
        <RecordContextProvider value={record}>
          <FormWithRedirect
            record={record}
            save={save}
            render={(formProps) => (
              <LibraryEditForm
                formProps={formProps}
                canEditPath
                canDelete={false}
              />
            )}
          />
        </RecordContextProvider>
      </SaveContextProvider>
    </TestContext>,
  )

const chooseAlbumGrouping = (optionText) => {
  fireEvent.mouseDown(
    screen.getByLabelText('resources.library.fields.pidAlbum'),
  )
  fireEvent.click(within(screen.getByRole('listbox')).getByText(optionText))
}

const dialogTitle = 'resources.library.messages.pidChangeTitle'

describe('LibraryEditForm', () => {
  afterEach(cleanup)

  it('saves directly when the PID config did not change', async () => {
    const save = vi.fn()
    renderForm(save)
    fireEvent.change(screen.getByLabelText(/resources.library.fields.name/), {
      target: { value: 'Jazz Renamed' },
    })
    fireEvent.click(screen.getByText('ra.action.save'))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect(screen.queryByText(dialogTitle)).not.toBeInTheDocument()
  })

  it('asks before saving a PID change, and Cancel keeps the edits', async () => {
    const save = vi.fn()
    renderForm(save)
    chooseAlbumGrouping('resources.library.pid.folder')
    fireEvent.click(screen.getByText('ra.action.save'))

    expect(await screen.findByText(dialogTitle)).toBeInTheDocument()
    expect(save).not.toHaveBeenCalled()

    fireEvent.click(screen.getByText('ra.action.cancel'))
    await waitFor(() =>
      expect(screen.queryByText(dialogTitle)).not.toBeInTheDocument(),
    )
    expect(save).not.toHaveBeenCalled()
    expect(screen.getByText('resources.library.pid.folder')).toBeInTheDocument()
  })

  it('saves the PID change after Confirm', async () => {
    const save = vi.fn()
    renderForm(save)
    chooseAlbumGrouping('resources.library.pid.folder')
    fireEvent.click(screen.getByText('ra.action.save'))
    fireEvent.click(await screen.findByText('ra.action.confirm'))

    await waitFor(() => expect(save).toHaveBeenCalled())
    expect(save.mock.calls[0][0]).toMatchObject({ pidAlbum: 'folder' })
  })

  it('pre-fills a Custom spec with the global spec', () => {
    renderForm(vi.fn())
    chooseAlbumGrouping('resources.library.pid.custom')
    expect(screen.getByLabelText(/resources.library.pid.spec/)).toHaveValue(
      config.pidAlbum,
    )
  })

  it('asks before saving when the form is submitted with Enter', async () => {
    const save = vi.fn()
    const { container } = renderForm(save)
    chooseAlbumGrouping('resources.library.pid.folder')
    fireEvent.submit(container.querySelector('form'))

    expect(await screen.findByText(dialogTitle)).toBeInTheDocument()
    expect(save).not.toHaveBeenCalled()
  })
})
