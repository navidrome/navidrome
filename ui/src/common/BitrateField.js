import React from 'react'
import PropTypes from 'prop-types'

const BitrateField = ({ record = {}, source }) => {
  return <span>{`${record[source]} kbps`}</span>
}

BitrateField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

BitrateField.defaultProps = {
  addLabel: true,
}

export default BitrateField
