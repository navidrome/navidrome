const ALBUM_MODE_GRID = 'ALBUM_GRID_MODE'
const ALBUM_MODE_LIST = 'ALBUM_LIST_MODE'
const selectViewMode = (mode) => ({ type: mode })

const albumViewReducer = (
  previousState = {
    mode: ALBUM_MODE_GRID,
  },
  payload
) => {
  const { type } = payload
  switch (type) {
    case ALBUM_MODE_GRID:
    case ALBUM_MODE_LIST:
      return { ...previousState, mode: type }
    default:
      return previousState
  }
}

export { ALBUM_MODE_LIST, ALBUM_MODE_GRID, albumViewReducer, selectViewMode }
