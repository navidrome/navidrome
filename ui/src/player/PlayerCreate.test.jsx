import * as React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import PlayerCreate from './PlayerCreate'
import ApiKeyInput from './ApiKeyInput'

const hooks = vi.hoisted(() => ({ forms: [] }))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    Create: ({ children }) => children,
    SimpleForm: (props) => {
      hooks.forms.push(props)
      return null
    },
  }
})

describe('PlayerCreate', () => {
  beforeEach(() => {
    hooks.forms = []
  })

  it('pre-fills one generated API key that survives re-renders', () => {
    const { rerender } = render(<PlayerCreate resource="player" />)
    rerender(<PlayerCreate resource="player" />)

    const [first, second] = hooks.forms.map((f) => f.initialValues)
    expect(hooks.forms).toHaveLength(2)
    expect(first.apiKey).toMatch(/^nds_[0-9A-Za-z]{22}$/)
    expect(second).toBe(first)
  })

  it('requires the API key', () => {
    render(<PlayerCreate resource="player" />)

    const input = React.Children.toArray(hooks.forms[0].children).find(
      (child) => child.type === ApiKeyInput,
    )
    expect(input.props.source).toBe('apiKey')
    expect(input.props.isCreate).toBe(true)
    expect(input.props.validate('')).toBeTruthy()
    expect(input.props.validate('nds_0123456789abcdefghijkl')).toBeUndefined()
  })
})
