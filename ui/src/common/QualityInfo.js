import React from 'react'
import PropTypes from 'prop-types'
import Chip from '@material-ui/core/Chip'
import config from '../config'

const llFormats = new Set(config.losslessFormats.split(','))
const placeholder = 'N/A'

export const QualityInfo = ({ record, ...rest }) => {
  let { suffix, bitRate } = record
  let info = placeholder

  if (suffix) {
    suffix = suffix.toUpperCase()
    info = suffix
    if (!llFormats.has(suffix)) {
      info += ' ' + bitRate
    }
  }

  return <Chip {...rest} variant="outlined" label={info} />
}

QualityInfo.propTypes = {
  record: PropTypes.object.isRequired,
  size: PropTypes.string,
}

QualityInfo.defaultProps = {
  size: 'small',
  record: {},
}
