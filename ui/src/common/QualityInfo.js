import React from 'react'
import PropTypes from 'prop-types'
import Chip from '@material-ui/core/Chip'
import config from '../config'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'
import Fade from '@material-ui/core/Fade'

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
  }
)

export const QualityInfo = ({ record, size, className, songTitleHover }) => {
  const classes = useStyle()
  let { suffix, bitRate } = record
  let info = placeholder

  if (suffix) {
    suffix = suffix.toUpperCase()
    info = suffix
    if (!llFormats.has(suffix)) {
      info += ' ' + bitRate
    }
  }

  return (
    <Fade in={songTitleHover} timeout={500}>
      <Chip
        className={clsx(classes.chip, className)}
        variant="outlined"
        size={size}
        label={info}
      />
    </Fade>
  )
}

QualityInfo.propTypes = {
  record: PropTypes.object.isRequired,
  size: PropTypes.string,
  className: PropTypes.string,
}

QualityInfo.defaultProps = {
  record: {},
  size: 'small',
}
