import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'

export const formatRange2 = (record, source) => {
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

export const RangeFieldDouble = ({ className, source1, source2, ...rest }) => {
  const record = useRecordContext(rest)
  if (formatRange2(record, source1) == formatRange2(record, source2)) {
      return <span className={className}>{formatRange(record, source1)}</span>
  } else {
  return <span className={className}>{"♫ " +
    formatRange2(record, source1) + " · □ " +
    formatRange2(record, source2)}</span>
  }
}

RangeFieldDouble.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

RangeFieldDouble.defaultProps = {
  addLabel: true,
}
