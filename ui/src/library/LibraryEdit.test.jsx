import * as React from 'react'
import { act, render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import LibraryEdit from './LibraryEdit'

const hooks = vi.hoisted(() => ({
  record: null,
  mutate: vi.fn(),
  notify: vi.fn(),
  redirect: vi.fn(),
  save: null,
  confirmOnConfirm: null,
  confirmOnClose: null,
}))

// `Edit` mirrors ra-ui-materialui's EditView: the fetched record is injected
// into the direct child only, never into the props of whoever renders <Edit>.
vi.mock('react-admin', () => ({
  Edit: ({ children }) =>
    React.cloneElement(React.Children.only(children), {
      record: hooks.record,
    }),
  FormWithRedirect: ({ record, save, render: renderForm }) => {
    hooks.save = save
    return renderForm({
      record,
      handleSubmit: () => {},
      pristine: false,
      saving: false,
    })
  },
  TextInput: ({ source }) => <input readOnly data-testid={`input-${source}`} />,
  BooleanInput: ({ source }) => (
    <input type="checkbox" readOnly data-testid={`input-${source}`} />
  ),
  required: () => () => null,
  SaveButton: () => <button data-testid="save-button">Save</button>,
  DateField: ({ source }) => <div data-testid={`date-${source}`} />,
  useTranslate: () => (key) => key,
  useMutation: () => [hooks.mutate],
  useNotify: () => hooks.notify,
  useRedirect: () => hooks.redirect,
  Toolbar: ({ children }) => <div data-testid="toolbar">{children}</div>,
  Confirm: ({ isOpen, onConfirm, onClose }) => {
    hooks.confirmOnConfirm = onConfirm
    hooks.confirmOnClose = onClose
    return isOpen ? <div data-testid="confirm-dialog" /> : null
  },
}))

vi.mock('./PIDAlbumInput', () => ({
  __esModule: true,
  default: () => <div data-testid="pid-album-input" />,
}))

vi.mock('./DeleteLibraryButton', () => ({
  __esModule: true,
  default: () => <button data-testid="delete-library-button">Delete</button>,
}))

vi.mock('../common', () => ({
  Title: () => <div data-testid="title" />,
  DocLink: ({ children }) => <a href="#doc">{children}</a>,
}))

vi.mock('@material-ui/core/styles', () => ({
  makeStyles: () => () => ({}),
}))

vi.mock('@material-ui/core', () => ({
  Typography: ({ children }) => <p>{children}</p>,
  Box: ({ children }) => <div>{children}</div>,
}))

describe('LibraryEdit save guard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    hooks.save = null
    hooks.confirmOnConfirm = null
    hooks.confirmOnClose = null
  })

  it('saves directly when the PID spec did not change', async () => {
    hooks.record = { id: '1', name: 'Library 1', pidAlbum: 'folder' }
    render(<LibraryEdit id="1" />)

    await act(async () => {
      hooks.save({ id: '1', name: 'Library 1 renamed', pidAlbum: 'folder' })
    })

    expect(hooks.mutate).toHaveBeenCalled()
    expect(screen.queryByTestId('confirm-dialog')).toBeNull()
  })

  it('opens the confirm dialog when pidAlbum changes from empty to folder', async () => {
    hooks.record = { id: '1', name: 'Library 1', pidAlbum: '' }
    render(<LibraryEdit id="1" />)

    await act(async () => {
      hooks.save({ id: '1', name: 'Library 1', pidAlbum: 'folder' })
    })

    expect(hooks.mutate).not.toHaveBeenCalled()
    expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument()
  })

  it('opens the confirm dialog when pidAlbum reverts from folder to empty', async () => {
    hooks.record = { id: '1', name: 'Library 1', pidAlbum: 'folder' }
    render(<LibraryEdit id="1" />)

    await act(async () => {
      hooks.save({ id: '1', name: 'Library 1', pidAlbum: '' })
    })

    expect(hooks.mutate).not.toHaveBeenCalled()
    expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument()
  })

  it('does not save on cancel, and leaves the form usable for another save', async () => {
    hooks.record = { id: '1', name: 'Library 1', pidAlbum: '' }
    render(<LibraryEdit id="1" />)

    await act(async () => {
      hooks.save({ id: '1', name: 'Library 1', pidAlbum: 'folder' })
    })
    expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument()

    await act(async () => {
      hooks.confirmOnClose()
    })

    expect(hooks.mutate).not.toHaveBeenCalled()
    expect(screen.queryByTestId('confirm-dialog')).toBeNull()

    await act(async () => {
      hooks.save({ id: '1', name: 'Library 1', pidAlbum: '' })
    })
    expect(hooks.mutate).toHaveBeenCalled()
  })

  it('calls the mutation when the dialog is confirmed', async () => {
    hooks.record = { id: '1', name: 'Library 1', pidAlbum: '' }
    render(<LibraryEdit id="1" />)

    await act(async () => {
      hooks.save({ id: '1', name: 'Library 1', pidAlbum: 'folder' })
    })

    await act(async () => {
      hooks.confirmOnConfirm()
    })

    expect(hooks.mutate).toHaveBeenCalled()
    expect(screen.queryByTestId('confirm-dialog')).toBeNull()
  })
})
