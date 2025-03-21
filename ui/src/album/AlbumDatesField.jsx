import { useRecordContext } from 'react-admin'
import { formatRange } from '../common/index.js'

const originalYearSymbol = '♫'
const releaseYearSymbol = '○'

export const AlbumDatesField = ({ className, ...rest }) => {
  const record = useRecordContext(rest)
  const releaseDate = record.releaseDate
  const releaseYear = releaseDate?.toString().substring(0, 4)
  const yearRange =
    formatRange(record, 'originalYear').toString() || record['maxYear']
  let label = yearRange

  if (yearRange !== releaseYear && releaseYear !== undefined) {
    label = `${originalYearSymbol} ${yearRange} · ${releaseYearSymbol} ${releaseYear}`
  }
  return <span className={className}>{label}</span>
}
