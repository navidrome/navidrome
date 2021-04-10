import { PAUSE_PLAYER, RESET_PLAYER } from '../actions'

export const playerReducer = (state = '', payload) => {
  const { type } = payload
  switch (type) {
    case PAUSE_PLAYER:
      return 'pause'
    case RESET_PLAYER:
      return ''
    default:
      return state
  }
}
