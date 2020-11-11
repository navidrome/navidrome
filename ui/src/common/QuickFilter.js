import React from 'react'
import { Chip, makeStyles } from '@material-ui/core'
import { useTranslate } from 'react-admin'

const useQuickFilterStyles = makeStyles((theme) => ({
  chip: {
    marginBottom: theme.spacing(1),
  },
}))

export const QuickFilter = ({ source, label }) => {
  const translate = useTranslate()
  const classes = useQuickFilterStyles()
  const lbl = label || `resources.song.fields.${source}`
  return <Chip className={classes.chip} label={translate(lbl)} />
}
