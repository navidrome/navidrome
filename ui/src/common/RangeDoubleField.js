import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'
import { formatRange } from '../common'

export const RangeDoubleField = ({
  className,
  source,
  symbol1,
  symbol2,
  separator,
  ...rest
}) => {
  const record = useRecordContext(rest)
  const yearRange = formatRange(record, source).toString()
  const releases = [record.releases]
  const releaseDate = [record.releaseDate]
  const releaseYear = releaseDate.toString().substring(0, 4)
  let subtitle = yearRange

  if (releases > 1) {
    subtitle = [
      [yearRange && symbol1, yearRange].join(' '),
      ['(', releases, ')))'].join(' '),
    ].join(separator)
  }

  if (
    yearRange !== releaseYear &&
    yearRange.length > 0 &&
    releaseYear.length > 0
  ) {
    subtitle = [
      [yearRange && symbol1, yearRange].join(' '),
      [symbol2, releaseYear].join(' '),
    ].join(separator)
  }

  return <span className={className}>{subtitle}</span>
}

RangeDoubleField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

RangeDoubleField.defaultProps = {
  addLabel: true,
}
