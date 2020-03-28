const ALBUM_GRID_MODE = 'ALBUM_GRID_MODE'
const ALBUM_LIST_MODE = 'ALBUM_LIST_MODE'

const selectViewMode = (mode) => ({ type: mode })

const albumViewReducer = (
  previousState = {
    mode: localStorage.getItem('albumViewMode') || ALBUM_LIST_MODE
  },
  payload
) => {
  const { type } = payload
  switch (type) {
    case ALBUM_GRID_MODE:
    case ALBUM_LIST_MODE:
      localStorage.setItem('albumViewMode', type)
      return { mode: type }
    default:
      return previousState
  }
}

export { ALBUM_LIST_MODE, ALBUM_GRID_MODE, albumViewReducer, selectViewMode }
