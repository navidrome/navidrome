export const ACTIVITY_SCAN_STATUS_UPD = 'ACTIVITY_SCAN_STATUS_UPD'

const actionsMap = { scanStatus: ACTIVITY_SCAN_STATUS_UPD }

export const updateScanStatus = (data) => {
  let type = actionsMap[data.name]
  if (!type) type = 'UNKNOWN'
  return {
    type,
    data: data.data,
  }
}
