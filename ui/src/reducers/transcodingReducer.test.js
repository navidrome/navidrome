import { describe, it, expect } from 'vitest'
import { transcodingReducer } from './transcodingReducer'
import { TRANSCODING_SET_PROFILE } from '../actions'

describe('transcodingReducer', () => {
  const initialState = { browserProfile: null }

  it('returns initial state', () => {
    expect(transcodingReducer(undefined, {})).toEqual(initialState)
  })

  it('handles TRANSCODING_SET_PROFILE', () => {
    const profile = {
      name: 'NavidromeUI',
      directPlayProfiles: [{ containers: ['mp3'] }],
    }
    const state = transcodingReducer(initialState, {
      type: TRANSCODING_SET_PROFILE,
      data: profile,
    })
    expect(state.browserProfile).toEqual(profile)
  })
})
