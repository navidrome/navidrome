import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'

export const formatDoubleRange = (record, source) => {
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

export const DoubleRangeField = ({ className, source1, source2, ...rest }) => {
  const record = useRecordContext(rest)
  return <span className={className}>{"♫ " +
    formatDoubleRange(record, source1) + " · □ "
    formatDoubleRange(record, source2)
  }</span>
}

DoubleRangeField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

DoubleRangeField.defaultProps = {
  addLabel: true,
}
