import React, { useMemo, useCallback } from 'react'
import { Card, CardContent, Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'
import { DurationField, SizeField } from '../common'

import AnchorMe from '../common/Linkify'
import clsx from 'clsx'

const useStyles = makeStyles(
  (theme) => ({
    container: {
      [theme.breakpoints.down('xs')]: {
        padding: '0.7em',
        minWidth: '24em',
      },
      [theme.breakpoints.up('sm')]: {
        padding: '1em',
        minWidth: '32em',
      },
    },
    details: {
      display: 'inline-block',
      verticalAlign: 'top',
      [theme.breakpoints.down('xs')]: {
        width: '14em',
      },
      [theme.breakpoints.up('sm')]: {
        width: '26em',
      },
      [theme.breakpoints.up('lg')]: {
        width: '38em',
      },
    },
    title: {
      whiteSpace: 'nowrap',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
    },
  }),
  {
    name: 'NDPlaylistDetails',
  },
)

const PlaylistComment = ({ record }) => {
  const classes = useStyles()
  const [expanded, setExpanded] = React.useState(false)

  const lines = record.comment.split('\n')
  const formatted = useMemo(() => {
    return lines.map((line, idx) => (
      <span key={record.id + '-comment-' + idx}>
        <AnchorMe text={line} />
        <br />
      </span>
    ))
  }, [lines, record.id])

  const handleExpandClick = useCallback(() => {
    setExpanded(!expanded)
  }, [expanded, setExpanded])

  return (
    <Collapse
      collapsedHeight={'2em'}
      in={expanded}
      timeout={'auto'}
      className={clsx(
        classes.commentBlock,
        lines.length > 1 && classes.pointerCursor,
      )}
    >
      <Typography variant={'h6'} onClick={handleExpandClick}>
        {formatted}
      </Typography>
    </Collapse>
  )
}

const PlaylistDetails = (props) => {
  const { record = {} } = props
  const translate = useTranslate()
  const classes = useStyles()

  return (
    <Card className={classes.container}>
      <CardContent className={classes.details}>
        <Typography variant="h5" className={classes.title}>
          {record.name || translate('ra.page.loading')}
        </Typography>
        <PlaylistComment record={record} />
        <Typography component="p">
          {record.songCount ? (
            <span>
              {record.songCount}{' '}
              {translate('resources.song.name', {
                smart_count: record.songCount,
              })}
              {' · '}
              <DurationField record={record} source={'duration'} />
              {' · '}
              <SizeField record={record} source={'size'} />
            </span>
          ) : (
            <span>&nbsp;</span>
          )}
        </Typography>
      </CardContent>
    </Card>
  )
}

export default PlaylistDetails
