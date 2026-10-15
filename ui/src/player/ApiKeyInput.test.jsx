import * as React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { Form } from 'react-final-form'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ApiKeyInput from './ApiKeyInput'

const hooks = vi.hoisted(() => ({ notify: vi.fn() }))
const KEY = 'nds_0123456789abcdefghijkl'
const KEY_FORMAT = /^nds_[0-9A-Za-z]{22}$/

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useNotify: () => hooks.notify,
    useTranslate: () => (key) => key,
  }
})

const renderInput = ({
  record,
  initialValues = {},
  isCreate = false,
  fullWidth,
}) => {
  let values
  const utils = render(
    <Form
      onSubmit={() => {}}
      initialValues={initialValues}
      render={({ values: v }) => {
        values = v
        return (
          <ApiKeyInput
            source="apiKey"
            record={record}
            isCreate={isCreate}
            fullWidth={fullWidth}
          />
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
    localStorage.setItem('role', 'regular')
  })

  it('shows a pending key with copy and regenerate on create', () => {
    const { values } = renderInput({
      record: {},
      isCreate: true,
      initialValues: { apiKey: KEY },
    })
    expect(screen.getByDisplayValue(KEY)).toBeInTheDocument()
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
    expect(values().apiKey).toMatch(KEY_FORMAT)
    expect(values().apiKey).not.toBe(KEY)
  })

  it('masks a saved key and lets the owner regenerate or revoke', () => {
    const { values } = renderInput({
      record: { id: 'p1', userId: 'owner', hasApiKey: true },
    })
    expect(screen.queryByDisplayValue(/^nds_/)).not.toBeInTheDocument()
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
    expect(values().apiKey).toMatch(KEY_FORMAT)
    expect(text('resources.player.message.apiKeyPending')).toBeInTheDocument()
  })

  it('lets an admin revoke but not set a key on another user player', () => {
    localStorage.setItem('role', 'admin')
    renderInput({ record: { id: 'p1', userId: 'someone', hasApiKey: true } })
    expect(
      text('resources.player.actions.regenerateApiKey'),
    ).not.toBeInTheDocument()
    expect(text('resources.player.actions.revokeApiKey')).toBeInTheDocument()
  })

  it('falls back to a prompt when the clipboard write fails', async () => {
    vi.stubGlobal('isSecureContext', true)
    vi.stubGlobal('prompt', vi.fn())
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
    })
    try {
      renderInput({
        record: {},
        isCreate: true,
        initialValues: { apiKey: KEY },
      })
      fireEvent.click(
        screen.getByRole('button', {
          name: 'resources.player.actions.copyApiKey',
        }),
      )
      await waitFor(() =>
        expect(window.prompt).toHaveBeenCalledWith(
          'message.shareCopyToClipboard',
          KEY,
        ),
      )
      expect(hooks.notify).not.toHaveBeenCalled()
    } finally {
      vi.unstubAllGlobals()
      delete navigator.clipboard
    }
  })

  it('shows a neutral message to an admin viewing another user player with no key', () => {
    localStorage.setItem('role', 'admin')
    renderInput({ record: { id: 'p1', userId: 'someone', hasApiKey: false } })
    expect(text('resources.player.message.apiKeyNoneOther')).toBeInTheDocument()
    expect(text('resources.player.message.apiKeyNone')).not.toBeInTheDocument()
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })

  it('shows no actions to another regular user', () => {
    renderInput({ record: { id: 'p1', userId: 'someone', hasApiKey: true } })
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })

  it('is not full width unless asked', () => {
    const record = { id: 'p1', userId: 'owner', hasApiKey: true }
    const { container, unmount } = renderInput({ record })
    expect(container.querySelector('.MuiFormControl-fullWidth')).toBeNull()
    unmount()

    const { container: wide } = renderInput({ record, fullWidth: true })
    expect(wide.querySelector('.MuiFormControl-fullWidth')).not.toBeNull()
  })
})
