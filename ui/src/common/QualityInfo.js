import React from 'react'
import Chip from '@material-ui/core/Chip'
import { LOSSLESS_FORMATS } from '../consts'

export const QualityInfo = ({record, ...rest}) => {
  let { suffix, bitRate } = record
  suffix = suffix.toUpperCase()
  let info = suffix
  if (!LOSSLESS_FORMATS.includes(suffix)) {
    info += ' ' + bitRate
  }

  return <Chip size="small" variant="outlined" label={info} />
}
