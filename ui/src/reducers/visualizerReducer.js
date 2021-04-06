import { SHOW_VISUALIZATION } from '../actions'

const initialState = {
  showVisualization: false,
}

export const visualizerReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case SHOW_VISUALIZATION:
      return {
        ...previousState,
        showVisualization: data,
      }
    default:
      return previousState
  }
}
