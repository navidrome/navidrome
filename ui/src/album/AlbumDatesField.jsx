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
  let label = yearRange

  if (releaseYear !== undefined && yearRange !== releaseYear) {
    label = `${originalYearSymbol} ${yearRange} · ${releaseYearSymbol} ${releaseYear}`
  }
  return <span className={className}>{label}</span>
}
