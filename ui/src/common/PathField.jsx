import PropTypes from 'prop-types'
import React from 'react'
import { useRecordContext } from 'react-admin'
import config from '../config'

export const PathField = (props) => {
  const record = useRecordContext(props)
  return (
    <span>
      {record.libraryPath}
      {config.separator}
      {record.path}
    </span>
  )
}

PathField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
}

PathField.defaultProps = {
  addLabel: true,
}
