import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
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
let mockVersion
vi.mock('react-admin', () => ({
  useTranslate: () => (x) => x,
  useGetIdentity: () => ({ loaded: true, identity: mockIdentity }),
  useVersion: () => mockVersion,
}))

const renderUserMenu = (identity) => {
  mockIdentity = identity
  return render(<UserMenu label="menu.settings" logout={<div>Logout</div>} />)
}

describe('<UserMenu />', () => {
  beforeEach(() => {
    localStorage.clear()
    mockVersion = 1
  })

  it('uses the uploaded avatar when there is an avatar tag', async () => {
    localStorage.setItem('avatarTag', 'abc123')
    renderUserMenu({ id: 'deluan', fullName: 'Deluan' })
    expect(await screen.findByRole('img')).toHaveAttribute(
      'src',
      expect.stringContaining('getAvatar'),
    )
  })

  it('falls back to the generic icon with no avatar and no gravatar', () => {
    renderUserMenu({ id: 'deluan', fullName: 'Deluan' })
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('uses the gravatar url when there is no avatar tag', () => {
    renderUserMenu({
      id: 'deluan',
      fullName: 'Deluan',
      avatar: 'https://gravatar/u1',
    })
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      'https://gravatar/u1',
    )
  })

  it('picks up an avatar uploaded during the session', async () => {
    const { rerender } = renderUserMenu({ id: 'deluan', fullName: 'Deluan' })
    expect(screen.queryByRole('img')).not.toBeInTheDocument()

    localStorage.setItem('avatarTag', 'newtag')
    mockVersion = 2
    rerender(<UserMenu label="menu.settings" logout={<div>Logout</div>} />)

    expect(await screen.findByRole('img')).toHaveAttribute(
      'src',
      expect.stringContaining('newtag'),
    )
  })

  it('goes back to the generic icon when the avatar is removed during the session', async () => {
    localStorage.setItem('avatarTag', 'abc123')
    const { rerender } = renderUserMenu({ id: 'deluan', fullName: 'Deluan' })
    expect(await screen.findByRole('img')).toBeInTheDocument()

    localStorage.removeItem('avatarTag')
    mockVersion = 2
    rerender(<UserMenu label="menu.settings" logout={<div>Logout</div>} />)

    await vi.waitFor(() =>
      expect(screen.queryByRole('img')).not.toBeInTheDocument(),
    )
  })
})
