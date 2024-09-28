import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'
import { formatRange } from './formatRange'

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
