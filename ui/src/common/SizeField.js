import React from 'react'
import PropTypes from 'prop-types'
import { formatBytes } from '../utils'

export const SizeField = ({ record = {}, source }) => {
  return <span>{formatBytes(record[source])}</span>
}

SizeField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

SizeField.defaultProps = {
  addLabel: true,
}
