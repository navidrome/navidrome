import { TRANSCODING_SET_PROFILE } from '../actions'

const initialState = {
  browserProfile: null,
}

export const transcodingReducer = (state = initialState, { type, data }) => {
  switch (type) {
    case TRANSCODING_SET_PROFILE:
      return { ...state, browserProfile: data }
    default:
      return state
  }
}
