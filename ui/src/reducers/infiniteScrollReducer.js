import { SET_INFINITE_SCROLL } from '../actions'

const initialState = {
  enabled: false,
}

export const infiniteScrollReducer = (
  previousState = initialState,
  payload,
) => {
  const { type, data } = payload
  switch (type) {
    case SET_INFINITE_SCROLL:
      return {
        ...previousState,
        enabled: data,
      }
    default:
      return previousState
  }
}
