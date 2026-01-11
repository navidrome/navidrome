import React from 'react'
import PropTypes from 'prop-types'
import { useRecordContext, useTranslate } from 'react-admin'
import { makeStyles, Tooltip, Typography } from '@material-ui/core'
import PeopleIcon from '@material-ui/icons/People'
import PlayCircleOutlineIcon from '@material-ui/icons/PlayCircleOutline'
import { formatCompactNumber, formatNumber } from '../utils'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: theme.spacing(1.5),
      color: theme.palette.text.secondary,
      fontSize: '0.875rem',
    },
    stat: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: theme.spacing(0.5),
    },
    icon: {
      fontSize: '1rem',
      opacity: 0.8,
    },
    value: {
      fontWeight: 500,
    },
  }),
  { name: 'NDPopularityField' },
)

export const PopularityField = ({
  listenersSource = 'lastfmListeners',
  playcountSource = 'lastfmPlaycount',
  showListeners = true,
  showPlaycount = true,
  ...rest
}) => {
  const record = useRecordContext(rest)
  const translate = useTranslate()
  const classes = useStyles()

  if (!record) return null

  const listeners = record[listenersSource]
  const playcount = record[playcountSource]

  // Don't render if no popularity data
  if ((!listeners || listeners === 0) && (!playcount || playcount === 0)) {
    return null
  }

  return (
    <span className={classes.root}>
      {showListeners && listeners > 0 && (
        <Tooltip
          title={`${formatNumber(listeners)} ${translate('resources.album.fields.lastfmListeners', { _: 'Last.fm listeners' })}`}
        >
          <span className={classes.stat}>
            <PeopleIcon className={classes.icon} />
            <Typography variant="body2" className={classes.value}>
              {formatCompactNumber(listeners)}
            </Typography>
          </span>
        </Tooltip>
      )}
      {showPlaycount && playcount > 0 && (
        <Tooltip
          title={`${formatNumber(playcount)} ${translate('resources.album.fields.lastfmPlaycount', { _: 'Last.fm plays' })}`}
        >
          <span className={classes.stat}>
            <PlayCircleOutlineIcon className={classes.icon} />
            <Typography variant="body2" className={classes.value}>
              {formatCompactNumber(playcount)}
            </Typography>
          </span>
        </Tooltip>
      )}
    </span>
  )
}

PopularityField.propTypes = {
  record: PropTypes.object,
  listenersSource: PropTypes.string,
  playcountSource: PropTypes.string,
  showListeners: PropTypes.bool,
  showPlaycount: PropTypes.bool,
}

export default PopularityField
