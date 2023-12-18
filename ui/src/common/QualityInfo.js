import React from 'react'
import PropTypes from 'prop-types'
import Chip from '@material-ui/core/Chip'
import config from '../config'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'

const llFormats = new Set(config.losslessFormats.split(','))
const placeholder = 'N/A'

const useStyle = makeStyles(
  (theme) => ({
    chip: {
      transform: 'scale(0.8)',
    },
  }),
  {
    name: 'NDQualityInfo',
  },
)

export const QualityInfo = ({ record, size, gainMode, preAmp, className }) => {
  const classes = useStyle()
  let { suffix, bitRate } = record
  let info = placeholder

  if (suffix) {
    suffix = suffix.toUpperCase()
    info = suffix
    if (!llFormats.has(suffix) && bitRate > 0) {
      info += ' ' + bitRate
    }
  }

  if (gainMode !== 'none') {
    info += ` (${
      (gainMode === 'album' ? record.albumGain : record.trackGain) + preAmp
    } dB)`
  }

  return (
    <Chip
      className={clsx(classes.chip, className)}
      variant="outlined"
      size={size}
      label={info}
    />
  )
}

QualityInfo.propTypes = {
  record: PropTypes.object.isRequired,
  size: PropTypes.string,
  className: PropTypes.string,
  gainMode: PropTypes.string,
}

QualityInfo.defaultProps = {
  record: {},
  size: 'small',
  gainMode: 'none',
}
