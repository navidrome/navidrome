import React from 'react'
import { Chip, makeStyles } from '@material-ui/core'
import { useTranslate } from 'react-admin'
import inflection from 'inflection'

const useQuickFilterStyles = makeStyles((theme) => ({
  chip: {
    marginBottom: theme.spacing(1),
  },
}))

export const QuickFilter = ({ source, resource, label, defaultValue }) => {
  const translate = useTranslate()
  const classes = useQuickFilterStyles()
  let lbl = label || source
  if (typeof lbl === 'string' || lbl instanceof String) {
    if (label) {
      lbl = translate(lbl, {
        _: inflection.humanize(inflection.underscore(lbl)),
      })
    } else {
      lbl = translate(`resources.${resource}.fields.${source}`, {
        _: inflection.humanize(inflection.underscore(source)),
      })
    }
  }
  return <Chip className={classes.chip} label={lbl} />
}
