export const EVENT_SCAN_STATUS = 'EVENT_SCAN_STATUS'
export const EVENT_SERVER_START = 'EVENT_SERVER_START'

const actionsMap = {
  scanStatus: EVENT_SCAN_STATUS,
  serverStart: EVENT_SERVER_START,
}

export const processEvent = (data) => {
  let type = actionsMap[data.name]
  if (!type) type = data.name
  return {
    type,
    data: data.data,
  }
}

export const scanStatusUpdate = (data) => ({
  type: EVENT_SCAN_STATUS,
  data: data,
})

export const serverDown = () => ({
  type: EVENT_SERVER_START,
  data: {},
})
