import React from 'react'
import {
  BulkActionsToolbar,
  ListToolbar,
  TextField,
  useVersion,
  useListContext,
  FunctionField,
} from 'react-admin'
import clsx from 'clsx'
import { useDispatch, useSelector } from 'react-redux'
import { Card, useMediaQuery, Typography } from '@material-ui/core'
import { useTheme } from '@material-ui/core/styles'
import { makeStyles } from '@material-ui/core/styles'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import get from 'lodash.get'
import { playTracks, togglePlayAction } from '../actions'
import {
  DurationField,
  SongBulkActions,
  SongContextMenu,
  SongDatagrid,
  SongDetails,
  SongTitleField,
} from '../common'
import { AddToPlaylistDialog } from '../dialogs'
import SongPlayIcon from '../common/SongPlayIcon'
import PlayingLight from '../icons/playing-light.gif'
import PlayingDark from '../icons/playing-dark.gif'
import PausedLight from '../icons/paused-light.png'
import PausedDark from '../icons/paused-dark.png'

const useStyles = makeStyles(
  (theme) => ({
    root: {},
    main: {
      display: 'flex',
    },
    content: {
      marginTop: 0,
      transition: theme.transitions.create('margin-top'),
      position: 'relative',
      flex: '1 1 auto',
      [theme.breakpoints.down('xs')]: {
        boxShadow: 'none',
      },
    },
    bulkActionsDisplayed: {
      marginTop: -theme.spacing(8),
      transition: theme.transitions.create('margin-top'),
    },
    actions: {
      zIndex: 2,
      display: 'flex',
      justifyContent: 'flex-end',
      flexWrap: 'wrap',
    },
    noResults: { padding: 20 },
    columnIcon: {
      marginLeft: '3px',
      marginTop: '-2px',
      verticalAlign: 'text-top',
    },
    toolbar: {
      justifyContent: 'flex-start',
    },
    row: {
      '&:hover': {
        '& $contextMenu': {
          visibility: 'visible',
        },
        '& $trackNoText': {
          display: 'none',
        },
        '& $playIcon': {
          display: 'block',
        },
        '& $icon': {
          display: 'none',
        },
        '& $pauseIcon': {
          display: 'block',
        },
      },
    },
    contextMenu: {
      visibility: (props) => (props.isDesktop ? 'hidden' : 'visible'),
    },
    trackNoText: {
      display: 'block',
      width: '24px',
    },
    icon: {
      display: 'block',
      width: '32px',
      height: '32px',
      verticalAlign: 'text-top',
      marginLeft: '-8px',
      marginTop: '-7px',
      paddingRight: '3px',
    },
    playIcon: {
      display: 'none',
    },
    pauseIcon: {
      display: 'none',
    },
  }),
  { name: 'RaList' }
)

const AlbumSongs = (props) => {
  const { data, ids } = props
  const theme = useTheme()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles({ isDesktop })
  const dispatch = useDispatch()
  const version = useVersion()
  const currentTrack = useSelector((state) => get(state, 'queue.current', {}))
  const currentId = currentTrack.trackId
  const paused = currentTrack.paused

  const Icon = () => {
    let icon
    if (paused) {
      icon = theme.palette.type === 'light' ? PausedLight : PausedDark
    } else {
      icon = theme.palette.type === 'light' ? PlayingLight : PlayingDark
    }
    return (
      <>
        <img
          src={icon}
          className={classes.icon}
          alt={paused ? 'paused' : 'playing'}
        />
        <SongPlayIcon
          isCurrent={true}
          onClick={() => {
            dispatch(togglePlayAction(false))
          }}
          className={classes.playIcon}
        />
      </>
    )
  }

  const handletrackNumber = (record) => {
    const isCurrent =
      currentId && (currentId === record.id || currentId === record.mediaFileId)
    if (isCurrent) {
      return <Icon />
    } else {
      return (
        <>
          <Typography className={classes.trackNoText} variant="subtitle2">
            {record.trackNumber}
          </Typography>
          <SongPlayIcon
            onClick={(id) => {
              if (id === record.id) {
                dispatch(togglePlayAction(false))
              } else {
                dispatch(playTracks(data, ids, id))
                dispatch(togglePlayAction(true))
              }
            }}
            record={record}
            className={classes.playIcon}
          />
        </>
      )
    }
  }

  return (
    <>
      <ListToolbar
        classes={{ toolbar: classes.toolbar }}
        actions={props.actions}
        {...props}
      />
      <div className={classes.main}>
        <Card
          className={clsx(classes.content, {
            [classes.bulkActionsDisplayed]: props.selectedIds.length > 0,
          })}
          key={version}
        >
          <BulkActionsToolbar {...props}>
            <SongBulkActions />
          </BulkActionsToolbar>
          <SongDatagrid
            expand={isXsmall ? null : <SongDetails />}
            rowClick={(id) => {
              if (id === currentId) {
                dispatch(togglePlayAction(false))
              } else {
                dispatch(playTracks(data, ids, id))
                dispatch(togglePlayAction(true))
              }
            }}
            {...props}
            hasBulkActions={true}
            showDiscSubtitles={true}
            contextAlwaysVisible={!isDesktop}
            classes={{ row: classes.row }}
          >
            {isDesktop && (
              <FunctionField
                source="trackNumber"
                sortBy="discNumber asc, trackNumber asc"
                label="#"
                sortable={false}
                render={handletrackNumber}
              />
            )}
            <SongTitleField
              source="title"
              sortable={false}
              showTrackNumbers={!isDesktop}
            />
            {isDesktop && <TextField source="artist" sortable={false} />}
            <DurationField source="duration" sortable={false} />
            <SongContextMenu
              source={'starred'}
              sortable={false}
              className={classes.contextMenu}
              label={
                <StarBorderIcon
                  fontSize={'small'}
                  className={classes.columnIcon}
                />
              }
            />
          </SongDatagrid>
        </Card>
      </div>
      <AddToPlaylistDialog />
    </>
  )
}

const SanitizedAlbumSongs = (props) => {
  const { loaded, loading, total, ...rest } = useListContext(props)
  return <>{loaded && <AlbumSongs {...rest} actions={props.actions} />}</>
}

export default SanitizedAlbumSongs
