import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockNotify = vi.fn()
const mockRefresh = vi.fn()

// Mock react-admin hooks
vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useUpdate: vi.fn(() => [vi.fn(), { loading: false }]),
    useNotify: vi.fn(() => mockNotify),
    useRefresh: vi.fn(() => mockRefresh),
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
    Button: ({ onClick, disabled, label, children }) => (
      <button onClick={onClick} disabled={disabled} data-testid="rescan-button">
        {children}
        {label}
      </button>
    ),
    TopToolbar: ({ children }) => (
      <div data-testid="top-toolbar">{children}</div>
    ),
    Datagrid: ({ children }) => (
      <table data-testid="datagrid">{children}</table>
    ),
    TextField: ({ source }) => <span data-testid={`text-${source}`} />,
  }
})

// Mock common components
vi.mock('../common', async () => {
  return {
    List: ({ children, actions, ...props }) => (
      <div data-testid="list">
        {actions}
        {children}
      </div>
    ),
    DateField: ({ source }) => <span data-testid={`date-${source}`} />,
    SimpleList: ({ primaryText, secondaryText }) => (
      <div data-testid="simple-list" />
    ),
    useResourceRefresh: vi.fn(),
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

// Mock httpClient
const mockHttpClient = vi.fn()
vi.mock('../dataProvider', () => ({
  httpClient: (...args) => mockHttpClient(...args),
}))

import PluginList from './PluginList'

describe('PluginList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockHttpClient.mockResolvedValue({})
  })

  it('renders the list component', () => {
    render(<PluginList />)
    expect(screen.getByTestId('list')).toBeInTheDocument()
  })

  it('renders the datagrid on desktop', () => {
    render(<PluginList />)
    expect(screen.getByTestId('datagrid')).toBeInTheDocument()
  })

  it('renders the rescan button', () => {
    render(<PluginList />)
    expect(screen.getByTestId('rescan-button')).toBeInTheDocument()
  })

  it('calls rescan endpoint when rescan button is clicked', async () => {
    render(<PluginList />)
    const rescanButton = screen.getByTestId('rescan-button')

    fireEvent.click(rescanButton)

    await waitFor(() => {
      expect(mockHttpClient).toHaveBeenCalledWith('/api/plugin/rescan', {
        method: 'POST',
      })
    })
  })

  it('calls refresh after successful rescan', async () => {
    render(<PluginList />)
    const rescanButton = screen.getByTestId('rescan-button')

    fireEvent.click(rescanButton)

    await waitFor(() => {
      expect(mockRefresh).toHaveBeenCalled()
    })
  })

  it('shows error notification on rescan failure', async () => {
    mockHttpClient.mockRejectedValue(new Error('Network error'))

    render(<PluginList />)
    const rescanButton = screen.getByTestId('rescan-button')

    fireEvent.click(rescanButton)

    await waitFor(() => {
      expect(mockNotify).toHaveBeenCalledWith('Network error', {
        type: 'warning',
      })
    })
  })
})
