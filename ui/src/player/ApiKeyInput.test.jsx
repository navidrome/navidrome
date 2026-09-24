import * as React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { Form } from 'react-final-form'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ApiKeyInput from './ApiKeyInput'

const hooks = vi.hoisted(() => ({ permissions: 'regular', notify: vi.fn() }))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    usePermissions: () => ({ permissions: hooks.permissions }),
    useNotify: () => hooks.notify,
    useTranslate: () => (key) => key,
  }
})

const renderInput = ({ record, initialValues = {}, isCreate = false }) => {
  let values
  const utils = render(
    <Form
      onSubmit={() => {}}
      initialValues={initialValues}
      render={({ values: v }) => {
        values = v
        return (
          <ApiKeyInput source="apiKey" record={record} isCreate={isCreate} />
        )
      }}
    />,
  )
  return { ...utils, values: () => values }
}

const text = (key) => screen.queryByText(key)

describe('ApiKeyInput', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('userId', 'owner')
    hooks.permissions = 'regular'
  })

  it('shows a pending key with copy and regenerate on create', () => {
    const { values } = renderInput({
      record: {},
      isCreate: true,
      initialValues: { apiKey: 'nav_0123456789abcdefghijkl' },
    })
    expect(
      screen.getByDisplayValue('nav_0123456789abcdefghijkl'),
    ).toBeInTheDocument()
    expect(text('resources.player.message.apiKeyPending')).toBeInTheDocument()
    expect(
      screen.getByRole('button', {
        name: 'resources.player.actions.copyApiKey',
      }),
    ).toBeInTheDocument()
    expect(
      text('resources.player.actions.revokeApiKey'),
    ).not.toBeInTheDocument()

    fireEvent.click(
      screen.getByText('resources.player.actions.regenerateApiKey'),
    )
    expect(values().apiKey).toMatch(/^nav_[0-9A-Za-z]{22}$/)
    expect(values().apiKey).not.toBe('nav_0123456789abcdefghijkl')
  })

  it('masks a saved key and lets the owner regenerate or revoke', () => {
    const { values } = renderInput({
      record: { id: 'p1', userId: 'owner', hasApiKey: true },
    })
    expect(screen.queryByDisplayValue(/^nav_/)).not.toBeInTheDocument()
    expect(text('resources.player.message.apiKeyActive')).toBeInTheDocument()
    expect(
      text('resources.player.actions.regenerateApiKey'),
    ).toBeInTheDocument()

    fireEvent.click(screen.getByText('resources.player.actions.revokeApiKey'))
    expect(values().apiKey).toBe('')
    expect(
      text('resources.player.message.apiKeyRevokePending'),
    ).toBeInTheDocument()
    expect(text('resources.player.actions.generateApiKey')).toBeInTheDocument()
  })

  it('lets the owner generate a key when there is none', () => {
    const { values } = renderInput({
      record: { id: 'p1', userId: 'owner', hasApiKey: false },
    })
    expect(values().apiKey).toBeUndefined()
    expect(text('resources.player.message.apiKeyNone')).toBeInTheDocument()

    fireEvent.click(screen.getByText('resources.player.actions.generateApiKey'))
    expect(values().apiKey).toMatch(/^nav_[0-9A-Za-z]{22}$/)
    expect(text('resources.player.message.apiKeyPending')).toBeInTheDocument()
  })

  it('lets an admin revoke but not set a key on another user player', () => {
    hooks.permissions = 'admin'
    renderInput({ record: { id: 'p1', userId: 'someone', hasApiKey: true } })
    expect(
      text('resources.player.actions.regenerateApiKey'),
    ).not.toBeInTheDocument()
    expect(text('resources.player.actions.revokeApiKey')).toBeInTheDocument()
  })

  it('shows no actions to another regular user', () => {
    renderInput({ record: { id: 'p1', userId: 'someone', hasApiKey: true } })
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })
})
