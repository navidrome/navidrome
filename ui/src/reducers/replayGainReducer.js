import { CHANGE_GAIN, CHANGE_PREAMP } from '../actions'
import config from '../config'

function initialState() {
  const state = {
    gainMode: localStorage.getItem('gainMode') || 'none',
    preAmp: getPreAmp(),
  }

  if (config.enableReplayGain && 'AudioContext' in window) {
    state.context = new AudioContext()
  }

  return state
}

const getPreAmp = () => {
  const storage = localStorage.getItem('preamp')

  if (storage === null) {
    return 0
  } else {
    const asFloat = parseFloat(storage)
    return isNaN(asFloat) ? 0 : asFloat
  }
}

export const replayGainReducer = (
  previousState = initialState(),
  { type, payload }
) => {
  switch (type) {
    case CHANGE_GAIN: {
      localStorage.setItem('gainMode', payload)
      return {
        ...previousState,
        gainMode: payload,
      }
    }
    case CHANGE_PREAMP: {
      const value = parseFloat(payload)
      if (isNaN(value)) {
        return previousState
      }
      localStorage.setItem('preAmp', payload)
      return {
        ...previousState,
        preAmp: value,
      }
    }
    default:
      return previousState
  }
}
