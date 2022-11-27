import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'

export const formatRange = (record, source) => {
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

export const RangeField = ({ className, source, ...rest }) => {
  const record = useRecordContext(rest)
  return <span className={className}>{formatRange(record, source)}</span>
}

RangeField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

RangeField.defaultProps = {
  addLabel: true,
}
