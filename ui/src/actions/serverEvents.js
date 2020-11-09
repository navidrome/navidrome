export const EVENT_SCAN_STATUS = 'ACTIVITY_SCAN_STATUS_UPD'

const actionsMap = { scanStatus: EVENT_SCAN_STATUS }

export const processEvent = (data) => {
  let type = actionsMap[data.name]
  if (!type) type = 'EVENT_UNKNOWN'
  return {
    type,
    data: data.data,
  }
}
