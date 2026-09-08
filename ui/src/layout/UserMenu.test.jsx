import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import UserMenu from './UserMenu'

vi.mock('../subsonic', () => ({
  default: {
    getAvatarUrl: vi.fn((username) => `/app/rest/getAvatar.view?u=${username}`),
  },
}))

vi.mock('react-redux', () => ({
  useDispatch: () => vi.fn(),
}))

let mockIdentity
vi.mock('react-admin', () => ({
  useTranslate: () => (x) => x,
  useGetIdentity: () => ({ loaded: true, identity: mockIdentity }),
}))

const renderUserMenu = (identity) => {
  mockIdentity = identity
  render(<UserMenu label="menu.settings" logout={<div>Logout</div>} />)
}

describe('<UserMenu />', () => {
  it('uses the uploaded avatar when the identity has an avatarTag', () => {
    renderUserMenu({ id: 'u1', username: 'deluan', avatarTag: 'abc123' })
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      expect.stringContaining('getAvatar'),
    )
  })

  it('falls back to the generic icon with no avatar and no gravatar', () => {
    renderUserMenu({ id: 'u1', username: 'deluan' })
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('uses the gravatar url when there is no avatarTag', () => {
    renderUserMenu({
      id: 'u1',
      username: 'deluan',
      avatar: 'https://gravatar/u1',
    })
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      'https://gravatar/u1',
    )
  })
})
