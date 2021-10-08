import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext } from 'react-admin'

export const BitrateField = ({ source, ...rest }) => {
  const record = useRecordContext(rest)
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
