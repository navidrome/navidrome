import * as React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import PlayerCreate from './PlayerCreate'

const hooks = vi.hoisted(() => ({ initialValues: [] }))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    Create: ({ children }) => children,
    SimpleForm: ({ initialValues }) => {
      hooks.initialValues.push(initialValues)
      return null
    },
  }
})

describe('PlayerCreate', () => {
  it('pre-fills one generated API key that survives re-renders', () => {
    const { rerender } = render(<PlayerCreate resource="player" />)
    rerender(<PlayerCreate resource="player" />)

    expect(hooks.initialValues).toHaveLength(2)
    expect(hooks.initialValues[0].apiKey).toMatch(/^nav_[0-9A-Za-z]{22}$/)
    expect(hooks.initialValues[1]).toBe(hooks.initialValues[0])
  })
})
