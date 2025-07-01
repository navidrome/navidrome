export const EVENT_SCAN_STATUS = 'scanStatus'
export const EVENT_SERVER_START = 'serverStart'
export const EVENT_REFRESH_RESOURCE = 'refreshResource'
export const EVENT_NOW_PLAYING_COUNT = 'nowPlayingCount'
export const EVENT_STREAM_RECONNECTED = 'streamReconnected'

export const processEvent = (type, data) => ({
  type,
  data: data,
})
export const scanStatusUpdate = (data) => ({
  type: EVENT_SCAN_STATUS,
  data: data,
})

export const nowPlayingCountUpdate = (data) => ({
  type: EVENT_NOW_PLAYING_COUNT,
  data: data,
})

export const serverDown = () => ({
  type: EVENT_SERVER_START,
  data: {},
})

export const streamReconnected = () => ({
  type: EVENT_STREAM_RECONNECTED,
  data: {},
})
