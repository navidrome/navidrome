import { formatBytes, formatDuration } from './formatters'

describe('formatBytes', () => {
  it('format bytes', () => {
    expect(formatBytes(0)).toEqual('0 Bytes')
    expect(formatBytes(1000)).toEqual('1000 Bytes')
    expect(formatBytes(1024)).toEqual('1 KB')
    expect(formatBytes(1024 * 1024)).toEqual('1 MB')
    expect(formatBytes(1024 * 1024 * 1024)).toEqual('1 GB')
    expect(formatBytes(1024 * 1024 * 1024 * 1024)).toEqual('1 TB')
  })
})

const day = 86400
const hour = 3600
const minute = 60

describe('formatDuration', () => {
  it('formats seconds', () => {
    expect(formatDuration(0)).toEqual('00:00')
    expect(formatDuration(59)).toEqual('00:59')
    expect(formatDuration(59.99)).toEqual('00:60')
  })

  it('formats days, hours and minutes', () => {
    expect(formatDuration(hour + minute + 1)).toEqual('01:01:01')
    expect(formatDuration(3 * day + 3 * hour + 7 * minute)).toEqual(
      '3:03:07:00'
    )
  })
})
