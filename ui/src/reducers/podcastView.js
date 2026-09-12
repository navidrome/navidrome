import { PODCAST_MODE_GRID, PODCAST_MODE_TABLE } from '../actions'

export const podcastViewReducer = (
  previousState = { grid: true },
  payload,
) => {
  const { type } = payload
  switch (type) {
    case PODCAST_MODE_GRID:
    case PODCAST_MODE_TABLE:
      return { ...previousState, grid: type === PODCAST_MODE_GRID }
    default:
      return previousState
  }
}
