import { formatBytes, formatDuration, formatFullDate } from './formatters'

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
    expect(formatDuration(59.99)).toEqual('01:00')
  })

  it('formats days, hours and minutes', () => {
    expect(formatDuration(hour + minute + 1)).toEqual('01:01:01')
    expect(formatDuration(3 * day + 3 * hour + 7 * minute)).toEqual(
      '3:03:07:00',
    )
    expect(formatDuration(day)).toEqual('1:00:00:00')
    expect(formatDuration(day + minute + 0.6)).toEqual('1:00:01:01')
  })
})

describe('formatFullDate', () => {
  it('format dates', () => {
    expect(formatFullDate('2011', 'en-US')).toEqual('2011')
    expect(formatFullDate('2011-06', 'en-US')).toEqual('Jun 2011')
    expect(formatFullDate('1985-01-01', 'en-US')).toEqual('Jan 1, 1985')
    expect(formatFullDate('199704')).toEqual('')
  })
})
