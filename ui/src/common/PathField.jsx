import PropTypes from 'prop-types'
import React from 'react'
import { usePermissions, useRecordContext } from 'react-admin'
import config from '../config'

export const PathField = (props) => {
  const record = useRecordContext(props)
  const { permissions } = usePermissions()
  let path = permissions === 'admin' ? record.libraryPath : ''

  if (path && path.endsWith(config.separator)) {
    path = `${path}${record.path}`
  } else {
    path = path ? `${path}${config.separator}${record.path}` : record.path
  }

  return <span>{path}</span>
}

PathField.propTypes = {
  record: PropTypes.object,
}
