import React from 'react'
import Chip from '@material-ui/core/Chip'

export const QualityInfo = (props) => {
  let { suffix, bitRate } = props.record
  suffix = suffix.toUpperCase()
  let info = suffix
  if (bitRate !== ''){
    info += ' ' + bitRate
  }

  return <Chip size="small" variant="outlined" label={info} />
}
