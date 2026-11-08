import { describe, it, expect, beforeEach } from 'vitest'
import { createMemoryHistory } from 'history'
import createAdminStore from './createAdminStore'
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

  it('restores the player play mode after a reload', () => {
    newStore().dispatch(setPlayMode('orderLoop'))

    expect(newStore().getState().player.mode).toBe('orderLoop')
  })
})
