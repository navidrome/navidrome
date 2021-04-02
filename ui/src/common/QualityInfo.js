import React from 'react'
import PropTypes from 'prop-types'
import Chip from '@material-ui/core/Chip'
import { LOSSLESS_FORMATS } from '../consts'

export const QualityInfo = ({ record, size, ...rest }) => {
  let { suffix, bitRate } = record
  suffix = suffix.toUpperCase()
  let info = suffix
  if (!LOSSLESS_FORMATS.includes(suffix)) {
    info += ' ' + bitRate
  }
  return <Chip {...rest} size={size} variant="outlined" label={info} />
}

QualityInfo.propTypes = {
  record: PropTypes.object.isRequired,
  size: PropTypes.string,
}

QualityInfo.defaultProps = {
  size: 'small',
}
