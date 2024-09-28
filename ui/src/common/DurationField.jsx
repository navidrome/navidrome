import React from 'react'
import PropTypes from 'prop-types'
import { formatDuration } from '../utils'
import { useRecordContext } from 'react-admin'

export const DurationField = ({ source, ...rest }) => {
  const record = useRecordContext(rest)
  try {
    return <span>{formatDuration(record[source])}</span>
  } catch (e) {
    // eslint-disable-next-line no-console
    console.log('Error in DurationField! Record:', record)
    return <span>00:00</span>
  }
}

DurationField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

DurationField.defaultProps = {
  addLabel: true,
}
