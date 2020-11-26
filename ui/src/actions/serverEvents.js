export const EVENT_SCAN_STATUS = 'scanStatus'
export const EVENT_SERVER_START = 'serverStart'

export const processEvent = (type, event) => {
  const data = JSON.parse(event)
  return {
    type,
    data: data,
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
