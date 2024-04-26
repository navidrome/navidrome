import { Card, CardContent, Typography } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'
import { CollapsibleComment, DurationField, SizeField } from '../common'

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
        <CollapsibleComment record={record} />
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
