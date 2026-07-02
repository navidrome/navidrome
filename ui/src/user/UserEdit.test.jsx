import * as React from 'react'
import { render, screen } from '@testing-library/react'
import UserEdit from './UserEdit'
import { describe, it, expect, vi } from 'vitest'

const defaultUser = {
  id: 'user1',
  userName: 'testuser',
  name: 'Test User',
  email: 'test@example.com',
  isAdmin: false,
  libraries: [
    { id: 1, name: 'Library 1', path: '/music1' },
    { id: 2, name: 'Library 2', path: '/music2' },
  ],
  lastLoginAt: '2023-01-01T12:00:00Z',
  lastAccessAt: '2023-01-02T12:00:00Z',
  updatedAt: '2023-01-03T12:00:00Z',
  createdAt: '2023-01-04T12:00:00Z',
}

const adminUser = {
  ...defaultUser,
  id: 'admin1',
  userName: 'admin',
  name: 'Admin User',
  isAdmin: true,
}

const hooks = vi.hoisted(() => ({
  save: null,
  mutate: vi.fn(),
  notify: vi.fn(),
  redirect: vi.fn(),
  refresh: vi.fn(),
}))

// Mock React-Admin completely with simpler implementations
vi.mock('react-admin', () => ({
  Edit: ({ children, title }) => (
    <div data-testid="edit-component">
      {title}
      {children}
    </div>
  ),
  SimpleForm: ({ children, save }) => {
    hooks.save = save
    return <form data-testid="simple-form">{children}</form>
  },
  TextInput: ({ source }) => <input data-testid={`text-input-${source}`} />,
  BooleanInput: ({ source }) => (
    <input type="checkbox" data-testid={`boolean-input-${source}`} />
  ),
  DateField: ({ source }) => (
    <div data-testid={`date-field-${source}`}>Date</div>
  ),
  PasswordInput: ({ source }) => (
    <input type="password" data-testid={`password-input-${source}`} />
  ),
  Toolbar: ({ children }) => <div data-testid="toolbar">{children}</div>,
  SaveButton: () => <button data-testid="save-button">Save</button>,
  FormDataConsumer: ({ children }) => children({ formData: {} }),
  Typography: ({ children }) => <p>{children}</p>,
  required: () => () => null,
  email: () => () => null,
  useMutation: () => [hooks.mutate],
  useNotify: () => hooks.notify,
  useRedirect: () => hooks.redirect,
  useRefresh: () => hooks.refresh,
  usePermissions: () => ({ permissions: 'admin' }),
  useTranslate: () => (key) => key,
}))

vi.mock('./LibrarySelectionField.jsx', () => ({
  LibrarySelectionField: () => <div data-testid="library-selection-field" />,
}))

vi.mock('./DeleteUserButton', () => ({
  __esModule: true,
  default: () => <button data-testid="delete-user-button">Delete</button>,
}))

vi.mock('../common', () => ({
  Title: ({ subTitle }) => <div data-testid="title">{subTitle}</div>,
}))

// Mock Material-UI
vi.mock('@material-ui/core/styles', () => ({
  makeStyles: () => () => ({}),
}))

vi.mock('@material-ui/core', () => ({
  Typography: ({ children }) => <p>{children}</p>,
}))

describe('<UserEdit />', () => {
  it('should render the user edit form', () => {
    render(<UserEdit id="user1" permissions="admin" />)

    // Check if the edit component renders
    expect(screen.getByTestId('edit-component')).toBeInTheDocument()
    expect(screen.getByTestId('simple-form')).toBeInTheDocument()
  })

  it('should render text inputs for admin users', () => {
    render(<UserEdit id="user1" permissions="admin" />)

    // Should render username input for admin
    expect(screen.getByTestId('text-input-userName')).toBeInTheDocument()
    expect(screen.getByTestId('text-input-name')).toBeInTheDocument()
    expect(screen.getByTestId('text-input-email')).toBeInTheDocument()
  })

  it('should render admin checkbox for admin permissions', () => {
    render(<UserEdit id="user1" permissions="admin" />)

    // Should render isAdmin checkbox for admin users
    expect(screen.getByTestId('boolean-input-isAdmin')).toBeInTheDocument()
  })

  it('should render date fields', () => {
    render(<UserEdit id="user1" permissions="admin" />)

    expect(screen.getByTestId('date-field-lastLoginAt')).toBeInTheDocument()
    expect(screen.getByTestId('date-field-lastAccessAt')).toBeInTheDocument()
    expect(screen.getByTestId('date-field-updatedAt')).toBeInTheDocument()
    expect(screen.getByTestId('date-field-createdAt')).toBeInTheDocument()
  })

  it('should not render username input for non-admin users', () => {
    render(<UserEdit id="user1" permissions="user" />)

    // Should not render username input for non-admin
    expect(screen.queryByTestId('text-input-userName')).not.toBeInTheDocument()
    // But should still render name and email
    expect(screen.getByTestId('text-input-name')).toBeInTheDocument()
    expect(screen.getByTestId('text-input-email')).toBeInTheDocument()
  })

  describe('save', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hooks.save = null
    })

    it('notifies success and redirects when the update succeeds', async () => {
      hooks.mutate.mockResolvedValue({ data: defaultUser })
      render(<UserEdit id="user1" permissions="admin" />)

      await hooks.save({ id: 'user1', name: 'New Name' })

      expect(hooks.notify).toHaveBeenCalledWith(
        'resources.user.notifications.updated',
        'info',
        { smart_count: 1 },
      )
      expect(hooks.redirect).toHaveBeenCalledWith('/user')
    })

    it('returns field errors when the update fails validation', async () => {
      const fieldErrors = { currentPassword: 'ra.validation.required' }
      hooks.mutate.mockRejectedValue({ body: { errors: fieldErrors } })
      render(<UserEdit id="user1" permissions="admin" />)

      const result = await hooks.save({ id: 'user1' })

      expect(result).toEqual(fieldErrors)
      expect(hooks.notify).not.toHaveBeenCalledWith(
        'resources.user.notifications.updated',
        'info',
        { smart_count: 1 },
      )
    })

    it('notifies an error when the update fails without field errors', async () => {
      hooks.mutate.mockRejectedValue(new Error('Forbidden'))
      render(<UserEdit id="user1" permissions="admin" />)

      await hooks.save({ id: 'user1' })

      expect(hooks.notify).toHaveBeenCalledWith('ra.page.error', 'warning')
      expect(hooks.redirect).not.toHaveBeenCalled()
    })

    it('notifies an error when the update rejects with a non-object error', async () => {
      hooks.mutate.mockRejectedValue(undefined)
      render(<UserEdit id="user1" permissions="admin" />)

      await hooks.save({ id: 'user1' })

      expect(hooks.notify).toHaveBeenCalledWith('ra.page.error', 'warning')
      expect(hooks.redirect).not.toHaveBeenCalled()
    })
  })
})
