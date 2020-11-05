import React from 'react'
import PropTypes from 'prop-types'

const DurationField = ({ record = {}, source }) => {
  try {
    return <span>{format(record[source])}</span>
  } catch (e) {
    console.log('Error in DurationField! Record:', record)
    return <span>00:00</span>
  }
}

const format = (d) => {
  const hours = Math.floor(d / 3600)
  const minutes = Math.floor(d / 60) % 60
  const seconds = d % 60
  return [hours, minutes, seconds]
    .map((v) => (v < 10 ? '0' + v : v))
    .filter((v, i) => v !== '00' || i > 0)
    .join(':')
}

DurationField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

DurationField.defaultProps = {
  addLabel: true,
}

export default DurationField
