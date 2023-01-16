import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'
import formatRange from '../common'

export const RangeFieldDouble = ({ className, source1, source2, ...rest }) => {
  const record = useRecordContext(rest)
  if (formatRange(record, source1) == formatRange(record, source2)) {
      return <span className={className}>{formatRange(record, source1)}</span>
  } else {
  return <span className={className}>{"♫ " +
    formatRange(record, source1) + " · ☼ " +
    formatRange(record, source2)}</span>
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
