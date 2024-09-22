import React, { useMemo } from 'react'
import PropTypes from 'prop-types'
import Chip from '@material-ui/core/Chip'
import config from '../config'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'
import { calculateGain } from '../utils/calculateReplayGain'

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
  let { suffix, bitRate, rgAlbumGain, rgAlbumPeak, rgTrackGain, rgTrackPeak } =
    record
  let info = placeholder

  if (suffix) {
    suffix = suffix.toUpperCase()
    info = suffix
    if (!llFormats.has(suffix) && bitRate > 0) {
      info += ' ' + bitRate
    }
  }

  const extra = useMemo(() => {
    if (gainMode !== 'none') {
      const gainValue = calculateGain(
        { gainMode, preAmp },
        { rgAlbumGain, rgAlbumPeak, rgTrackGain, rgTrackPeak },
      )
      // convert normalized gain (after peak) back to dB
      const toDb = (Math.log10(gainValue) * 20).toFixed(2)
      return ` (${toDb} dB)`
    }

    return ''
  }, [gainMode, preAmp, rgAlbumGain, rgAlbumPeak, rgTrackGain, rgTrackPeak])

  return (
    <Chip
      className={clsx(classes.chip, className)}
      variant="outlined"
      size={size}
      label={`${info}${extra}`}
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
