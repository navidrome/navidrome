import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { ReadOnlyField } from './ReadOnlyField'

describe('<ReadOnlyField>', () => {
  const record = { id: '1', client: 'NavidromeUI' }

  const renderField = (props) =>
    render(
      <ReadOnlyField
        record={record}
        resource="player"
        source="client"
        {...props}
      />,
    )

  it('shows the record value with the translated field label', () => {
    renderField()
    const input = screen.getByLabelText('resources.player.fields.client')
    expect(input).toHaveValue('NavidromeUI')
  })

  it('cannot be edited or reached with the Tab key', () => {
    renderField()
    const input = screen.getByRole('textbox')
    expect(input).toHaveAttribute('readonly')
    expect(input).toHaveAttribute('tabindex', '-1')
  })

  it('shows an empty value when the record has no value', () => {
    renderField({ source: 'userName' })
    expect(screen.getByRole('textbox')).toHaveValue('')
  })

  it('uses an explicit label when given', () => {
    renderField({ label: 'Custom' })
    expect(screen.getByLabelText('Custom')).toBeInTheDocument()
  })

  it('exposes a theme-overridable class name', () => {
    const { container } = renderField()
    expect(container.firstChild.className).toMatch(/NDReadOnlyField-root/)
  })
})
