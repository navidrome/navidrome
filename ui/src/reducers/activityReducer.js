import { ACTIVITY_SCAN_STATUS_UPD } from '../actions'

export const activityReducer = (
  previousState = {
    scanStatus: { scanning: false, count: 0 },
  },
  payload
) => {
  const { type, data } = payload
  switch (type) {
    case ACTIVITY_SCAN_STATUS_UPD:
      return { ...previousState, scanStatus: data }
    default:
      return previousState
  }
}
