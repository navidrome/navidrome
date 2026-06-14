import * as React from 'react'
import { render, screen, cleanup } from '@testing-library/react'
import { describe, it, expect, afterEach, vi, beforeEach } from 'vitest'
import { useRecordContext } from 'react-admin'
import { TranscodingInput } from './PlayerEdit'

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useRecordContext: vi.fn(),
    // Render the inputs as simple stand-ins so we can read their props.
    ReferenceInput: ({ children, variant }) => (
      <div data-testid="reference-input" data-variant={variant || ''}>
        {children}
      </div>
    ),
    SelectInput: ({ helperText }) => (
      <div data-testid="select-input" data-helpertext={helperText || ''} />
    ),
  }
})

describe('<TranscodingInput />', () => {
  beforeEach(() => {
    useRecordContext.mockReset()
  })
  afterEach(cleanup)

  it('shows helper text for the NavidromeUI player', () => {
    useRecordContext.mockReturnValue({ client: 'NavidromeUI' })
    render(<TranscodingInput />)
    expect(screen.getByTestId('select-input').dataset.helpertext).toBe(
      'resources.player.helperTexts.transcodingId',
    )
  })

  it('shows no helper text for other clients', () => {
    useRecordContext.mockReturnValue({ client: 'DSub' })
    render(<TranscodingInput />)
    expect(screen.getByTestId('select-input').dataset.helpertext).toBe('')
  })

  it('forwards the form variant injected by SimpleForm to the input', () => {
    useRecordContext.mockReturnValue({ client: 'DSub' })
    render(<TranscodingInput variant="outlined" />)
    expect(screen.getByTestId('reference-input').dataset.variant).toBe(
      'outlined',
    )
  })
})
