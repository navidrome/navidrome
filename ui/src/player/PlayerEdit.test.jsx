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
    ReferenceInput: ({ children, helperText }) => (
      <div data-testid="reference-input" data-helpertext={helperText || ''}>
        {children}
      </div>
    ),
    SelectInput: () => <div data-testid="select-input" />,
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
    expect(screen.getByTestId('reference-input').dataset.helpertext).toBe(
      'resources.player.helperTexts.transcodingId',
    )
  })

  it('shows no helper text for other clients', () => {
    useRecordContext.mockReturnValue({ client: 'DSub' })
    render(<TranscodingInput />)
    expect(screen.getByTestId('reference-input').dataset.helpertext).toBe('')
  })
})
