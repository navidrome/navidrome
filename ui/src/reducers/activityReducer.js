import {
  EVENT_REFRESH_RESOURCE,
  EVENT_SCAN_STATUS,
  EVENT_SERVER_START,
} from '../actions'
import config from '../config'

const initialState = {
  scanStatus: { scanning: false, folderCount: 0, count: 0, error: '' },
  serverStart: { version: config.version },
}

export const activityReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case EVENT_SCAN_STATUS:
      return { ...previousState, scanStatus: data }
    case EVENT_SERVER_START:
      return {
        ...previousState,
        serverStart: {
          startTime: data.startTime && Date.parse(data.startTime),
          version: data.version,
        },
      }
    case EVENT_REFRESH_RESOURCE:
      return {
        ...previousState,
        refresh: {
          lastReceived: Date.now(),
          resources: data,
        },
      }
    default:
      return previousState
  }
}
