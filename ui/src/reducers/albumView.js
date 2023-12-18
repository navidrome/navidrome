import { ALBUM_MODE_GRID, ALBUM_MODE_TABLE } from '../actions'

export const albumViewReducer = (
  previousState = {
    grid: true,
  },
  payload,
) => {
  const { type } = payload
  switch (type) {
    case ALBUM_MODE_GRID:
    case ALBUM_MODE_TABLE:
      return { ...previousState, grid: type === ALBUM_MODE_GRID }
    default:
      return previousState
  }
}
