import React from 'react'
import PropTypes from 'prop-types'
import { formatRange } from '../utils'

export const RangeField = ({ className, record = {}, source }) => {
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
