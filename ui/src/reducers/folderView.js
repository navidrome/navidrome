import { FOLDER_MODE_GRID, FOLDER_MODE_TABLE } from '../actions'

export const folderViewReducer = (
  previousState = {
    grid: false, // Default to table for folders as it's more standard for directories
  },
  payload,
) => {
  const { type } = payload
  switch (type) {
    case FOLDER_MODE_GRID:
    case FOLDER_MODE_TABLE:
      return { ...previousState, grid: type === FOLDER_MODE_GRID }
    default:
      return previousState
  }
}
