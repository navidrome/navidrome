import * as React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import PlayerEdit from './PlayerEdit'

const hooks = vi.hoisted(() => ({ editProps: null }))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    Edit: (props) => {
      hooks.editProps = props
      return null
    },
  }
})

describe('PlayerEdit', () => {
  // An optimistic or undoable save would put the new key in react-admin's cache
  it('saves pessimistically', () => {
    render(<PlayerEdit resource="player" id="p1" />)
    expect(hooks.editProps.mutationMode).toBe('pessimistic')
  })
})
