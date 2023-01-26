export const CHANGE_GAIN = 'CHANGE_GAIN'
export const CHANGE_PREAMP = 'CHANGE_PREAMP'

export const changeGain = (gain) => ({
  type: CHANGE_GAIN,
  payload: gain,
})

export const changePreamp = (preamp) => ({
  type: CHANGE_PREAMP,
  payload: preamp,
})
