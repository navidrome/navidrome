export const EVENT_SCAN_STATUS = 'EVENT_SCAN_STATUS'

const actionsMap = { scanStatus: EVENT_SCAN_STATUS }

export const processEvent = (data) => {
  let type = actionsMap[data.name]
  if (!type) type = data.name
  return {
    type,
    data: data.data,
  }
}

export const scanStatusUpdate = (data) =>
  processEvent({
    name: EVENT_SCAN_STATUS,
    data: data,
  })
