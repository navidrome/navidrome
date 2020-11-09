import { EVENT_SCAN_STATUS } from '../actions'

export const activityReducer = (
  previousState = {
    scanStatus: { scanning: false, count: 0 },
  },
  payload
) => {
  const { type, data } = payload
  switch (type) {
    case EVENT_SCAN_STATUS:
      return { ...previousState, scanStatus: data }
    default:
      return previousState
  }
}
