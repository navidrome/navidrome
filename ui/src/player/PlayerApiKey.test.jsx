import * as React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import PlayerApiKey from './PlayerApiKey'

const hooks = vi.hoisted(() => ({
  record: null,
  permissions: 'regular',
  dataProvider: {
    generatePlayerApiKey: vi.fn(),
    revokePlayerApiKey: vi.fn(),
  },
  notify: vi.fn(),
  refresh: vi.fn(),
}))

vi.mock('react-admin', () => ({
  Button: ({ label, onClick, children }) => (
    <button onClick={onClick}>
      {children}
      {label}
    </button>
  ),
  Confirm: ({ isOpen, title, onConfirm }) =>
    isOpen ? (
      <div>
        {title}
        <button onClick={onConfirm}>confirm</button>
      </div>
    ) : null,
  useRecordContext: () => hooks.record,
  usePermissions: () => ({ permissions: hooks.permissions }),
  useDataProvider: () => hooks.dataProvider,
  useNotify: () => hooks.notify,
  useRefresh: () => hooks.refresh,
  useTranslate: () => (key) => key,
}))

describe('PlayerApiKey', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('userId', 'owner')
    hooks.permissions = 'regular'
  })

  it('lets the owner generate a key and shows it once', async () => {
    hooks.record = { id: 'p1', userId: 'owner', hasApiKey: false }
    hooks.dataProvider.generatePlayerApiKey.mockResolvedValue({
      data: { apiKey: 'nav_secret' },
    })
    render(<PlayerApiKey />)

    fireEvent.click(screen.getByText('resources.player.actions.generateApiKey'))

    await waitFor(() =>
      expect(screen.getByDisplayValue('nav_secret')).toBeInTheDocument(),
    )
    expect(hooks.dataProvider.generatePlayerApiKey).toHaveBeenCalledWith('p1')
  })

  it('shows regenerate and revoke to the owner when a key exists', () => {
    hooks.record = { id: 'p1', userId: 'owner', hasApiKey: true }
    render(<PlayerApiKey />)

    expect(
      screen.getByText('resources.player.actions.regenerateApiKey'),
    ).toBeInTheDocument()
    expect(
      screen.getByText('resources.player.actions.revokeApiKey'),
    ).toBeInTheDocument()
  })

  it('lets an admin revoke but not generate for another user', async () => {
    hooks.permissions = 'admin'
    hooks.record = { id: 'p1', userId: 'someone-else', hasApiKey: true }
    hooks.dataProvider.revokePlayerApiKey.mockResolvedValue({
      data: { id: 'p1' },
    })
    render(<PlayerApiKey />)

    expect(
      screen.queryByText('resources.player.actions.regenerateApiKey'),
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('resources.player.actions.revokeApiKey'))
    fireEvent.click(screen.getByText('confirm'))

    await waitFor(() =>
      expect(hooks.dataProvider.revokePlayerApiKey).toHaveBeenCalledWith('p1'),
    )
  })

  it('shows no buttons to a non-owner regular user', () => {
    hooks.record = { id: 'p1', userId: 'someone-else', hasApiKey: true }
    render(<PlayerApiKey />)

    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })
})
