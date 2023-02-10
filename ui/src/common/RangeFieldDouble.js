import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'
import { formatRange } from '../common'

export const RangeFieldDouble = ({ className, source, symbol1, symbol2, separator, ...rest }) => {
  const record = useRecordContext(rest)
  const yearRange = formatRange(record, source).toString()
  const editions = [record.editions]
  const releaseDate = [record.releaseDate]
  const releaseYear = releaseDate.toString().substring(0,4)
  let subtitle = yearRange

  if (editions > 1) {
    subtitle = [
      (yearRange && symbol1) + yearRange,
      '( ' + editions + ' )))'
      ].join(separator)
   }

   if ((yearRange != releaseYear) && (yearRange.length > 0) && (releaseYear.length > 0)) {
    subtitle = [
      (yearRange && symbol1) + yearRange,
      symbol2 + releaseYear
      ].join(separator)
   }

  return <span className={className}>
    {subtitle}
    </span>
}

RangeFieldDouble.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

RangeFieldDouble.defaultProps = {
  addLabel: true,
}
