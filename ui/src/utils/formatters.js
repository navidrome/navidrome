export const formatBytes = (bytes, decimals = 2) => {
  if (bytes === 0) return '0 Bytes'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']

  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

export const formatRange = (record, source) => {
  const nameCapitalized = source.charAt(0).toUpperCase() + source.slice(1)
  const min = record[`min${nameCapitalized}`]
  const max = record[`max${nameCapitalized}`]
  let range = []
  if (min) {
    range.push(min)
  }
  if (max && max !== min) {
    range.push(max)
  }
  return range.join('-')
}

export const formatDuration = (d) => {
  const hours = Math.floor(d / 3600)
  const minutes = Math.floor(d / 60) % 60
  const seconds = d % 60
  return [hours, minutes, seconds]
    .map((v) => Math.round(v).toString())
    .map((v) => (v.length !== 2 ? '0' + v : v))
    .filter((v, i) => v !== '00' || i > 0)
    .join(':')
}
