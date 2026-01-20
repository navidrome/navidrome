import React from 'react'
import { describe, it, expect, vi } from 'vitest'
import { render } from '@testing-library/react'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import { SchemaConfigEditor } from './SchemaConfigEditor'

const theme = createTheme()

// JSONForms requires Redux
const mockStore = createStore(() => ({}))

const renderWithProviders = (component) => {
  return render(
    <Provider store={mockStore}>
      <ThemeProvider theme={theme}>{component}</ThemeProvider>
    </Provider>,
  )
}

describe('SchemaConfigEditor', () => {
  const basicSchema = {
    type: 'object',
    properties: {
      name: {
        type: 'string',
        title: 'Name',
      },
      enabled: {
        type: 'boolean',
        title: 'Enabled',
      },
    },
  }

  it('renders nothing when schema is null', () => {
    const { container } = renderWithProviders(
      <SchemaConfigEditor schema={null} data={{}} onChange={vi.fn()} />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders the component wrapper with valid schema', () => {
    const { container } = renderWithProviders(
      <SchemaConfigEditor schema={basicSchema} data={{}} onChange={vi.fn()} />,
    )
    // Check that the wrapper div is rendered (class name is generated)
    expect(
      container.querySelector('[class*="NDSchemaConfigEditor-root"]'),
    ).toBeTruthy()
  })

  it('calls onChange on initial render', () => {
    const onChange = vi.fn()
    renderWithProviders(
      <SchemaConfigEditor
        schema={basicSchema}
        data={{ name: 'Test' }}
        onChange={onChange}
      />,
    )

    // JSONForms calls onChange on initial render with initial state
    expect(onChange).toHaveBeenCalled()
  })

  it('passes data and errors to onChange callback', () => {
    const onChange = vi.fn()
    const initialData = { name: 'Test Value' }

    renderWithProviders(
      <SchemaConfigEditor
        schema={basicSchema}
        data={initialData}
        onChange={onChange}
      />,
    )

    // Check that onChange was called with data and errors
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Test Value' }),
      expect.any(Array),
    )
  })
})
