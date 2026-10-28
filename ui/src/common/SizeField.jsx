import React from 'react'
import PropTypes from 'prop-types'
import { formatBytes } from '../utils'
import { useRecordContext } from 'react-admin'
import { useTheme } from '@mui/material'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles((theme) => ({
  root: {
    display: 'inline-block',
  },
}))

export const SizeField = ({ source, ...rest }) => {
  const classes = useStyles()
  const record = useRecordContext(rest)
  if (!record) return null
  return (
    <span className={classes.root}>
      {record[source] ? formatBytes(record[source]) : '0 MB'}
    </span>
  )
}

SizeField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

SizeField.defaultProps = {
  addLabel: true,
}
