import React from 'react'
import PropTypes from 'prop-types'
import { formatDuration } from '../utils'

export const DurationField = ({ record = {}, source }) => {
  try {
    return <span>{formatDuration(record[source])}</span>
  } catch (e) {
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
