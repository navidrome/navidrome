import * as React from 'react'
import { TestContext } from 'ra-test'
import { DataProviderContext } from 'react-admin'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { describe, afterEach, it, expect, vi } from 'vitest'
import { QuickConnectDialog } from './QuickConnectDialog'

const finamp = { appName: 'Finamp', appVersion: '1.0.0', deviceName: 'Pixel 7' }

const renderDialog = (dataProvider, onClose = vi.fn()) => {
  render(
    <DataProviderContext.Provider value={dataProvider}>
      <TestContext initialState={{ admin: { ui: { optimistic: false } } }}>
        <QuickConnectDialog open={true} onClose={onClose} />
      </TestContext>
    </DataProviderContext.Provider>,
  )
  return onClose
}

const enterCode = (code) => {
  fireEvent.change(screen.getByRole('textbox'), { target: { value: code } })
  fireEvent.click(screen.getByText('menu.quickConnect.continue'))
}

describe('QuickConnectDialog', () => {
  afterEach(cleanup)

  it('shows the device before approving the code', async () => {
    const dataProvider = {
      lookupQuickConnect: vi.fn().mockResolvedValue({ data: finamp }),
      authorizeQuickConnect: vi.fn().mockResolvedValue({ data: finamp }),
    }
    const onClose = renderDialog(dataProvider)

    enterCode('123 456')
    await screen.findByText('menu.quickConnect.approve')
    expect(dataProvider.lookupQuickConnect.mock.calls[0][0]).toBe('123 456')
    expect(dataProvider.authorizeQuickConnect).not.toHaveBeenCalled()

    fireEvent.click(screen.getByText('menu.quickConnect.approve'))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(dataProvider.authorizeQuickConnect.mock.calls[0][0]).toBe('123 456')
  })

  it('goes back to the code input', async () => {
    const dataProvider = {
      lookupQuickConnect: vi.fn().mockResolvedValue({ data: finamp }),
      authorizeQuickConnect: vi.fn(),
    }
    renderDialog(dataProvider)

    enterCode('123456')
    await screen.findByText('menu.quickConnect.approve')
    fireEvent.click(screen.getByText('ra.action.back'))

    expect(screen.getByRole('textbox')).toHaveValue('123456')
    expect(dataProvider.authorizeQuickConnect).not.toHaveBeenCalled()
  })

  it('stays on the code input when the code is unknown', async () => {
    const dataProvider = {
      lookupQuickConnect: vi.fn().mockRejectedValue({ status: 404 }),
      authorizeQuickConnect: vi.fn(),
    }
    const onClose = renderDialog(dataProvider)

    enterCode('000000')
    await waitFor(() =>
      expect(dataProvider.lookupQuickConnect).toHaveBeenCalled(),
    )
    expect(screen.getByRole('textbox')).toBeInTheDocument()
    expect(screen.queryByText('menu.quickConnect.approve')).toBeNull()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('returns to the code input when approval fails', async () => {
    const dataProvider = {
      lookupQuickConnect: vi.fn().mockResolvedValue({ data: finamp }),
      authorizeQuickConnect: vi.fn().mockRejectedValue({ status: 409 }),
    }
    const onClose = renderDialog(dataProvider)

    enterCode('123456')
    fireEvent.click(await screen.findByText('menu.quickConnect.approve'))

    await screen.findByRole('textbox')
    expect(onClose).not.toHaveBeenCalled()
  })

  it('disables Continue until a code is typed', () => {
    renderDialog({ lookupQuickConnect: vi.fn() })
    expect(
      screen.getByText('menu.quickConnect.continue').closest('button'),
    ).toBeDisabled()
  })
})
