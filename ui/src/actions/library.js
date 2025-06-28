export const SET_SELECTED_LIBRARIES = 'SET_SELECTED_LIBRARIES'
export const SET_USER_LIBRARIES = 'SET_USER_LIBRARIES'

export const setSelectedLibraries = (libraryIds) => ({
  type: SET_SELECTED_LIBRARIES,
  data: libraryIds,
})

export const setUserLibraries = (libraries) => ({
  type: SET_USER_LIBRARIES,
  data: libraries,
})
