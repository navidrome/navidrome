import { useRecordContext } from 'react-admin'
import { formatRange } from '../common/index.js'

const originalYearSymbol = '♫'
const releaseYearSymbol = '○'

export const AlbumDatesField = ({ className, ...rest }) => {
  const record = useRecordContext(rest)
  const releaseDate = record.releaseDate
  const releaseYear = releaseDate?.toString().substring(0, 4)
  const yearRange =
    formatRange(record, 'originalYear') || record['maxYear']?.toString()

  // Don't show anything if the year starts with "0"
  if (yearRange === '0' || releaseYear?.startsWith('0')) {
    return null
  }

  let label = yearRange

  if (releaseYear !== undefined && yearRange !== releaseYear) {
    label = `${originalYearSymbol} ${yearRange} · ${releaseYearSymbol} ${releaseYear}`
  }
  return <span className={className}>{label}</span>
}
