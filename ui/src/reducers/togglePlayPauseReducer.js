import { TOGGLE_PLAY } from '../actions'

const initialState = {
  playing: false,
}

export const togglePlayPauseReducer = (
  previousState = initialState,
  payload
) => {
  const { type, data } = payload
  switch (type) {
    case TOGGLE_PLAY:
      return { ...previousState, playing: data }
    default:
      return previousState
  }
}
