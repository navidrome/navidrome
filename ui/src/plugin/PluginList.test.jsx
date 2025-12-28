import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock react-admin hooks
vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useUpdate: vi.fn(() => [vi.fn(), { loading: false }]),
    useNotify: vi.fn(() => vi.fn()),
    useRefresh: vi.fn(() => vi.fn()),
    useTranslate: vi.fn(() => (key) => key),
    useResourceContext: vi.fn(() => 'plugin'),
    useRecordContext: vi.fn(() => ({
      id: 'test-plugin',
      manifest: JSON.stringify({
        name: 'Test Plugin',
        version: '1.0.0',
        description: 'Test plugin',
      }),
      enabled: true,
      lastError: null,
    })),
    Datagrid: ({ children }) => (
      <table data-testid="datagrid">{children}</table>
    ),
    TextField: ({ source }) => <span data-testid={`text-${source}`} />,
  }
})

// Mock common components
vi.mock('../common', async () => {
  return {
    List: ({ children, ...props }) => (
      <div data-testid="list" {...props}>
        {children}
      </div>
    ),
    DateField: ({ source }) => <span data-testid={`date-${source}`} />,
    SimpleList: ({ primaryText, secondaryText }) => (
      <div data-testid="simple-list" />
    ),
  }
})

// Mock Material-UI
vi.mock('@material-ui/core', async () => {
  const actual = await vi.importActual('@material-ui/core')
  return {
    ...actual,
    useMediaQuery: vi.fn(() => false),
  }
})

// Mock ToggleEnabledSwitch
vi.mock('./ToggleEnabledSwitch', () => ({
  default: () => <span data-testid="toggle-switch" />,
}))

import PluginList from './PluginList'

describe('PluginList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the list component', () => {
    render(<PluginList />)
    expect(screen.getByTestId('list')).toBeInTheDocument()
  })

  it('renders the datagrid on desktop', () => {
    render(<PluginList />)
    expect(screen.getByTestId('datagrid')).toBeInTheDocument()
  })
})
