import * as React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import UserEdit from './UserEdit'
import config from '../config'
import { describe, it, expect, vi, afterEach } from 'vitest'

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
  permissions: 'admin',
  record: null,
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
  usePermissions: () => ({ permissions: hooks.permissions }),
  useRecordContext: () => hooks.record,
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
  ImageUploadOverlay: ({ canEdit, messages, onImageChange }) =>
    canEdit ? (
      <>
        <button
          aria-label={messages.uploadLabel}
          onClick={() => onImageChange(true)}
        >
          upload
        </button>
        <button
          aria-label={messages.removeLabel}
          onClick={() => onImageChange(false)}
        >
          remove
        </button>
      </>
    ) : null,
}))

vi.mock('../subsonic', () => ({
  default: {
    getAvatarUrl: (username) => `/rest/getAvatar?username=${username}`,
  },
}))

// Mock Material-UI
vi.mock('@material-ui/core/styles', () => ({
  makeStyles: () => () => ({}),
}))

vi.mock('@material-ui/core', () => ({
  Typography: ({ children }) => <p>{children}</p>,
  Avatar: ({ src, alt }) => <img data-testid="avatar" src={src} alt={alt} />,
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

  it('should render the scrobble filter input', () => {
    render(<UserEdit id="user1" permissions="admin" />)

    expect(screen.getByTestId('text-input-scrobbleFilter')).toBeInTheDocument()
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

  describe('avatar upload', () => {
    afterEach(() => {
      localStorage.clear()
      hooks.record = null
      hooks.permissions = 'admin'
    })

    const renderUserEdit = (
      record,
      { isMyself = false, role = 'user' } = {},
    ) => {
      localStorage.setItem('userId', isMyself ? record.id : 'someone-else')
      hooks.record = record
      hooks.permissions = role
      return render(<UserEdit id={record.id} permissions={role} />)
    }

    it('shows the avatar upload control for the user themselves', () => {
      config.enableUserAvatarUpload = true
      renderUserEdit({ id: 'u1', userName: 'deluan' }, { isMyself: true })
      expect(screen.getByLabelText('message.uploadAvatar')).toBeInTheDocument()
    })

    it('stores a new avatar tag when the user uploads their own avatar', () => {
      config.enableUserAvatarUpload = true
      renderUserEdit({ id: 'u1', userName: 'deluan' }, { isMyself: true })

      fireEvent.click(screen.getByLabelText('message.uploadAvatar'))

      expect(localStorage.getItem('avatarTag')).toBeTruthy()
    })

    it('removes the avatar tag when the user removes their own avatar', () => {
      config.enableUserAvatarUpload = true
      localStorage.setItem('avatarTag', 'oldtag')
      renderUserEdit(
        { id: 'u1', userName: 'deluan', uploadedImage: 'u1_deluan.png' },
        { isMyself: true },
      )

      fireEvent.click(screen.getByLabelText('message.removeAvatar'))

      expect(localStorage.getItem('avatarTag')).toBeNull()
    })

    it('does not touch the admin own tag when editing another user', () => {
      config.enableUserAvatarUpload = true
      renderUserEdit(
        { id: 'u1', userName: 'deluan', uploadedImage: 'u1_deluan.png' },
        { isMyself: false, role: 'admin' },
      )
      localStorage.setItem('avatarTag', 'mytag')

      fireEvent.click(screen.getByLabelText('message.uploadAvatar'))
      fireEvent.click(screen.getByLabelText('message.removeAvatar'))

      expect(localStorage.getItem('avatarTag')).toEqual('mytag')
    })

    it('hides the control when the feature is off and the viewer is not an admin', () => {
      config.enableUserAvatarUpload = false
      renderUserEdit(
        { id: 'u1', userName: 'deluan' },
        { isMyself: true, role: 'regular' },
      )
      expect(
        screen.queryByLabelText('message.uploadAvatar'),
      ).not.toBeInTheDocument()
    })
  })
})
