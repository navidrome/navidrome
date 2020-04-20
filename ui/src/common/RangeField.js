import React from 'react'
import PropTypes from 'prop-types'

const formatRange = (record, source) => {
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

const RangeField = ({ record = {}, source }) => {
  return <span>{formatRange(record, source)}</span>
}

RangeField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

RangeField.defaultProps = {
  addLabel: true,
}

export { formatRange }
export default RangeField
