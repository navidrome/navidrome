import { describe, it, expect } from 'vitest'
import { expandInfoDialogReducer } from './dialogReducer'
import { EXTENDED_INFO_OPEN, EXTENDED_INFO_CLOSE } from '../actions'

describe('expandInfoDialogReducer', () => {
  it('stores the record and resource on EXTENDED_INFO_OPEN', () => {
    const record = { id: 'al1', name: 'Album' }
    const result = expandInfoDialogReducer(
      { open: false, record: undefined, resource: undefined },
      { type: EXTENDED_INFO_OPEN, record, resource: 'album' },
    )

    expect(result).toEqual({ open: true, record, resource: 'album' })
  })

  it('clears the record and resource on EXTENDED_INFO_CLOSE', () => {
    const previousState = {
      open: true,
      record: { id: 'al1', name: 'Album' },
      resource: 'album',
    }
    const result = expandInfoDialogReducer(previousState, {
      type: EXTENDED_INFO_CLOSE,
    })

    expect(result).toEqual({
      open: false,
      record: undefined,
      resource: undefined,
    })
  })

  it('returns previous state for unknown action', () => {
    const previousState = {
      open: false,
      record: undefined,
      resource: undefined,
    }
    const result = expandInfoDialogReducer(previousState, {
      type: 'UNKNOWN_ACTION',
    })

    expect(result).toBe(previousState)
  })
})
