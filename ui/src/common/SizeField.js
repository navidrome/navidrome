import React from 'react'
import PropTypes from 'prop-types'
import { formatBytes } from '../utils'
import { makeStyles } from '@material-ui/core'

const useStyles = makeStyles((theme) => ({
  demo: {
    display: 'inline-block',
  },
}))

export const SizeField = ({ record = {}, source }) => {
  const classes = useStyles()

  return <span className={classes.demo}>{formatBytes(record[source])}</span>
}

SizeField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

SizeField.defaultProps = {
  addLabel: true,
}
