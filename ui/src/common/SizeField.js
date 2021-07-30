import React from 'react'
import PropTypes from 'prop-types'
import { formatBytes } from '../utils'
import { useRecordContext } from 'react-admin'

export const SizeField = ({ source, ...rest }) => {
  const record = useRecordContext(rest)
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
