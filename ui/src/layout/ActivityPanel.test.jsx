import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { describe, it, beforeEach } from 'vitest'

import ActivityPanel from './ActivityPanel'
import { activityReducer } from '../reducers'
import config from '../config'
import subsonic from '../subsonic'

vi.mock('../subsonic', () => ({
  default: {
    getScanStatus: vi.fn(() =>
      Promise.resolve({
        json: {
          'subsonic-response': {
            status: 'ok',
            scanStatus: { error: 'Scan failed' },
          },
        },
      }),
    ),
    startScan: vi.fn(),
  },
}))

describe('<ActivityPanel />', () => {
  let store

  beforeEach(() => {
    store = createStore(combineReducers({ activity: activityReducer }), {
      activity: {
        scanStatus: {
          scanning: false,
          folderCount: 0,
          count: 0,
          error: 'Scan failed',
          elapsedTime: 0,
        },
        serverStart: { version: config.version, startTime: Date.now() },
      },
    })
  })

  it('clears the error icon after opening the panel', () => {
    render(
      <Provider store={store}>
        <ActivityPanel />
      </Provider>,
    )

    const button = screen.getByRole('button')
    expect(screen.getByTestId('activity-error-icon')).toBeInTheDocument()

    fireEvent.click(button)

    expect(screen.getByTestId('activity-ok-icon')).toBeInTheDocument()
    expect(screen.getByText('Scan failed')).toBeInTheDocument()
  })
})
