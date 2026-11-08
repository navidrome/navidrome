import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createMemoryHistory } from 'history'
import createAdminStore from './createAdminStore'
import { loadState } from './persistState'
import { playerReducer } from '../reducers'
import { setPlayMode } from '../actions'

const newStore = () =>
  createAdminStore({
    authProvider: {},
    dataProvider: {},
    history: createMemoryHistory(),
    customReducers: { player: playerReducer },
  })

describe('createAdminStore', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('restores the player play mode after a reload', async () => {
    newStore().dispatch(setPlayMode('orderLoop'))
    await vi.waitFor(() => expect(loadState()?.player?.mode).toBe('orderLoop'))

    expect(newStore().getState().player.mode).toBe('orderLoop')
  })
})
