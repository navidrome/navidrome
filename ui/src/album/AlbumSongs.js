import React from 'react'
import {
  BulkActionsToolbar,
  ListToolbar,
  TextField,
  useVersion,
  useListContext,
} from 'react-admin'
import clsx from 'clsx'
import { useDispatch } from 'react-redux'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import { playTracks } from '../actions'
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
      },
    },
    contextMenu: {
      visibility: (props) => (props.isDesktop ? 'hidden' : 'visible'),
    },
  }),
  { name: 'RaList' }
)

const AlbumSongs = (props) => {
  const { data, ids } = props
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles({ isDesktop })
  const dispatch = useDispatch()
  const version = useVersion()
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
            rowClick={(id) => dispatch(playTracks(data, ids, id))}
            {...props}
            hasBulkActions={true}
            showDiscSubtitles={true}
            contextAlwaysVisible={!isDesktop}
            classes={{ row: classes.row }}
          >
            {isDesktop && (
              <TextField
                source="trackNumber"
                sortBy="discNumber asc, trackNumber asc"
                label="#"
                sortable={false}
              />
            )}
            {isDesktop && (
              <SongPlayIcon
                onClick={(id) => dispatch(playTracks(data, ids, id))}
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
