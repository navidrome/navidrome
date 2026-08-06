import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import AppPasswordManager from './AppPasswordManager.jsx'

const notify = vi.fn()
vi.mock('react-admin', () => ({
  useNotify: () => notify,
}))

const httpClient = vi.fn()
vi.mock('../dataProvider/httpClient', () => ({
  default: (...args) => httpClient(...args),
}))

vi.mock('../consts', () => ({
  REST_URL: '/api',
}))

const password = (overrides = {}) => ({
  id: 'ap1',
  name: 'DSub',
  createdAt: '2026-05-01T10:00:00Z',
  lastUsedAt: null,
  expiresAt: null,
  ...overrides,
})

describe('<AppPasswordManager />', () => {
  beforeEach(() => {
    httpClient.mockReset()
    notify.mockReset()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  it('shows the empty state when the user has no app passwords', async () => {
    httpClient.mockResolvedValueOnce({ json: [] })

    render(<AppPasswordManager userId="u1" />)

    expect(await screen.findByText('No app passwords yet.')).toBeInTheDocument()
    expect(httpClient).toHaveBeenCalledWith('/api/user/u1/app-password')
  })

  it('renders a row for each existing app password', async () => {
    httpClient.mockResolvedValueOnce({
      json: [password({ name: 'DSub' }), password({ id: 'ap2', name: 'Symfonium' })],
    })

    render(<AppPasswordManager userId="u1" />)

    expect(await screen.findByText('DSub')).toBeInTheDocument()
    expect(screen.getByText('Symfonium')).toBeInTheDocument()
  })

  it('creates a password and reveals the secret exactly once', async () => {
    httpClient
      .mockResolvedValueOnce({ json: [] }) // initial list
      .mockResolvedValueOnce({ json: { id: 'ap1', name: 'CLI', secret: 's3cret' } }) // create
      .mockResolvedValueOnce({ json: [password({ name: 'CLI' })] }) // refresh

    render(<AppPasswordManager userId="u1" />)
    await screen.findByText('No app passwords yet.')

    fireEvent.click(screen.getByRole('button', { name: /generate new/i }))
    const nameInput = screen.getAllByRole('textbox')[0]
    fireEvent.change(nameInput, { target: { value: 'CLI' } })
    fireEvent.click(screen.getByRole('button', { name: /^generate$/i }))

    expect(await screen.findByDisplayValue('s3cret')).toBeInTheDocument()
    expect(httpClient).toHaveBeenCalledWith('/api/user/u1/app-password', {
      method: 'POST',
      body: JSON.stringify({ name: 'CLI' }),
    })
  })

  it('deletes a password only after the user confirms', async () => {
    httpClient
      .mockResolvedValueOnce({ json: [password({ id: 'ap1', name: 'DSub' })] }) // initial list
      .mockResolvedValueOnce({ json: {} }) // delete
      .mockResolvedValueOnce({ json: [] }) // refresh

    render(<AppPasswordManager userId="u1" />)
    const row = await screen.findByText('DSub')

    fireEvent.click(row.closest('tr').querySelector('button'))

    await waitFor(() =>
      expect(httpClient).toHaveBeenCalledWith('/api/user/u1/app-password/ap1', {
        method: 'DELETE',
      }),
    )
  })

  it('does not delete when the user cancels the confirmation', async () => {
    window.confirm.mockReturnValue(false)
    httpClient.mockResolvedValueOnce({
      json: [password({ id: 'ap1', name: 'DSub' })],
    })

    render(<AppPasswordManager userId="u1" />)
    const row = await screen.findByText('DSub')

    fireEvent.click(row.closest('tr').querySelector('button'))

    expect(httpClient).toHaveBeenCalledTimes(1) // only the initial list
  })
})
