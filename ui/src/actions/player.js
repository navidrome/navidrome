export const PAUSE_PLAYER = 'PAUSE_PLAYER'
export const RESET_PLAYER = 'RESET_PLAYER'

export const pausePlayer = () => {
  return {
    type: PAUSE_PLAYER,
  }
}

export const resetPlayer = () => {
  return {
    type: RESET_PLAYER,
  }
}
