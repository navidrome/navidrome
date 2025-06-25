import {
  formatBytes,
  formatDuration,
  formatDuration2,
  formatFullDate,
  formatNumber,
  formatShortDuration,
} from './formatters'

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

describe('formatShortDuration', () => {
  // Convert seconds to nanoseconds for the tests
  const toNs = (seconds) => seconds * 1e9

  it('formats less than a second', () => {
    expect(formatShortDuration(toNs(0.5))).toEqual('<1s')
    expect(formatShortDuration(toNs(0))).toEqual('<1s')
  })

  it('formats seconds', () => {
    expect(formatShortDuration(toNs(1))).toEqual('1s')
    expect(formatShortDuration(toNs(59))).toEqual('59s')
  })

  it('formats minutes and seconds', () => {
    expect(formatShortDuration(toNs(60))).toEqual('1m0s')
    expect(formatShortDuration(toNs(90))).toEqual('1m30s')
    expect(formatShortDuration(toNs(59 * 60 + 59))).toEqual('59m59s')
  })

  it('formats hours and minutes', () => {
    expect(formatShortDuration(toNs(3600))).toEqual('1h0m')
    expect(formatShortDuration(toNs(3600 + 30 * 60))).toEqual('1h30m')
    expect(formatShortDuration(toNs(24 * 3600 - 1))).toEqual('23h59m')
  })
})

describe('formatDuration2', () => {
  it('handles null and undefined values', () => {
    expect(formatDuration2(null)).toEqual('0s')
    expect(formatDuration2(undefined)).toEqual('0s')
  })

  it('handles negative values', () => {
    expect(formatDuration2(-10)).toEqual('0s')
    expect(formatDuration2(-1)).toEqual('0s')
  })

  it('formats zero seconds', () => {
    expect(formatDuration2(0)).toEqual('0s')
  })

  it('formats seconds only', () => {
    expect(formatDuration2(1)).toEqual('1s')
    expect(formatDuration2(30)).toEqual('30s')
    expect(formatDuration2(59)).toEqual('59s')
  })

  it('formats minutes and seconds', () => {
    expect(formatDuration2(60)).toEqual('1m')
    expect(formatDuration2(90)).toEqual('1m 30s')
    expect(formatDuration2(119)).toEqual('1m 59s')
    expect(formatDuration2(120)).toEqual('2m')
  })

  it('formats hours, minutes and seconds', () => {
    expect(formatDuration2(3600)).toEqual('1h')
    expect(formatDuration2(3661)).toEqual('1h 1m 1s')
    expect(formatDuration2(7200)).toEqual('2h')
    expect(formatDuration2(7260)).toEqual('2h 1m')
    expect(formatDuration2(7261)).toEqual('2h 1m 1s')
  })

  it('handles decimal values by flooring', () => {
    expect(formatDuration2(59.9)).toEqual('59s')
    expect(formatDuration2(60.1)).toEqual('1m')
    expect(formatDuration2(3600.9)).toEqual('1h')
  })

  it('formats days with maximum 3 levels (d h m)', () => {
    expect(formatDuration2(86400)).toEqual('1d')
    expect(formatDuration2(86461)).toEqual('1d 1m') // seconds dropped when days present
    expect(formatDuration2(90061)).toEqual('1d 1h 1m') // seconds dropped when days present
    expect(formatDuration2(172800)).toEqual('2d')
    expect(formatDuration2(176400)).toEqual('2d 1h')
    expect(formatDuration2(176460)).toEqual('2d 1h 1m')
    expect(formatDuration2(176461)).toEqual('2d 1h 1m') // seconds dropped when days present
  })
})

describe('formatNumber', () => {
  it('handles null and undefined values', () => {
    expect(formatNumber(null)).toEqual('0')
    expect(formatNumber(undefined)).toEqual('0')
  })

  it('formats integers', () => {
    expect(formatNumber(0)).toEqual('0')
    expect(formatNumber(1)).toEqual('1')
    expect(formatNumber(123)).toEqual('123')
    expect(formatNumber(1000)).toEqual('1,000')
    expect(formatNumber(1234567)).toEqual('1,234,567')
  })

  it('formats decimal numbers', () => {
    expect(formatNumber(123.45)).toEqual('123.45')
    expect(formatNumber(1234.567)).toEqual('1,234.567')
  })

  it('formats negative numbers', () => {
    expect(formatNumber(-123)).toEqual('-123')
    expect(formatNumber(-1234)).toEqual('-1,234')
    expect(formatNumber(-123.45)).toEqual('-123.45')
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
