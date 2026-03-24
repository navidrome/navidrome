import { SET_SELECTED_LIBRARIES, SET_USER_LIBRARIES } from '../actions'

const initialState = {
  userLibraries: [],
  selectedLibraries: [], // Empty means "all accessible libraries"
}

export const libraryReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case SET_USER_LIBRARIES: {
      const newUserLibraryIds = data.map((lib) => lib.id)

      // Validate and filter selected libraries to only include IDs that exist in new user libraries
      const validatedSelection = previousState.selectedLibraries.filter((id) =>
        newUserLibraryIds.includes(id),
      )

      // Determine the final selection:
      // 1. If first time setting libraries (no previous user libraries), select all
      // 2. If user now has only one library, reset to empty (no filter needed)
      // 3. Otherwise, use validated selection (may be empty if all previous selections were invalid)
      let finalSelection
      if (
        previousState.selectedLibraries.length === 0 &&
        previousState.userLibraries.length === 0
      ) {
        // First time: select all libraries
        finalSelection = newUserLibraryIds
      } else if (newUserLibraryIds.length === 1) {
        // Single library: reset selection (empty means "all accessible")
        finalSelection = []
      } else {
        // Multiple libraries: use validated selection
        finalSelection = validatedSelection
      }

      return {
        ...previousState,
        userLibraries: data,
        selectedLibraries: finalSelection,
      }
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
