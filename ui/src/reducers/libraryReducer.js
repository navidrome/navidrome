import { SET_SELECTED_LIBRARIES, SET_USER_LIBRARIES } from '../actions'

const initialState = {
  userLibraries: [],
  selectedLibraries: [], // Empty means "all accessible libraries"
}

export const libraryReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case SET_USER_LIBRARIES:
      return {
        ...previousState,
        userLibraries: data,
        // If this is the first time setting user libraries and no selection exists,
        // default to all libraries
        selectedLibraries:
          previousState.selectedLibraries.length === 0 &&
          previousState.userLibraries.length === 0
            ? data.map((lib) => lib.id)
            : previousState.selectedLibraries,
      }
    case SET_SELECTED_LIBRARIES:
      return {
        ...previousState,
        selectedLibraries: data,
      }
    default:
      return previousState
  }
}
