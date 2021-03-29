import React, { useCallback } from 'react'
import {
  BulkActionsToolbar,
  ListToolbar,
  TextField,
  useRefresh,
  useDataProvider,
  useNotify,
  useVersion,
  useListContext,
} from 'react-admin'
import clsx from 'clsx'
import { useDispatch } from 'react-redux'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import ReactDragListView from 'react-drag-listview'
import {
  DurationField,
  SongDetails,
  SongContextMenu,
  SongDatagrid,
  SongTitleField,
} from '../common'
import { AddToPlaylistDialog } from '../dialogs'
import { AlbumLinkField } from '../song/AlbumLinkField'
import { playTracks } from '../actions'
import PlaylistSongBulkActions from './PlaylistSongBulkActions'
import { QualityInfo } from '../common/QualityInfo'

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

const ReorderableList = ({ readOnly, children, ...rest }) => {
  if (readOnly) {
    return children
  }
  return <ReactDragListView {...rest}>{children}</ReactDragListView>
}

const PlaylistSongs = ({ playlistId, readOnly, ...props }) => {
  const { data, ids, onUnselectItems } = props
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles({ isDesktop })
  const dispatch = useDispatch()
  const dataProvider = useDataProvider()
  const refresh = useRefresh()
  const notify = useNotify()
  const version = useVersion()

  const onAddToPlaylist = useCallback(
    (pls) => {
      if (pls.id === playlistId) {
        refresh()
      }
    },
    [playlistId, refresh]
  )

  const reorder = useCallback(
    (playlistId, id, newPos) => {
      dataProvider
        .update('playlistTrack', {
          id,
          data: { insert_before: newPos },
          filter: { playlist_id: playlistId },
        })
        .then(() => {
          refresh()
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        })
    },
    [dataProvider, notify, refresh]
  )

  const handleDragEnd = useCallback(
    (from, to) => {
      const toId = ids[to]
      const fromId = ids[from]
      reorder(playlistId, fromId, toId)
    },
    [playlistId, reorder, ids]
  )

  return (
    <>
      <ListToolbar
        classes={{ toolbar: classes.toolbar }}
        filters={props.filters}
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
            <PlaylistSongBulkActions
              playlistId={playlistId}
              onUnselectItems={onUnselectItems}
            />
          </BulkActionsToolbar>
          <ReorderableList
            readOnly={readOnly}
            onDragEnd={handleDragEnd}
            nodeSelector={'tr'}
          >
            <SongDatagrid
              expand={!isXsmall && <SongDetails />}
              rowClick={(id) => dispatch(playTracks(data, ids, id))}
              {...props}
              hasBulkActions={true}
              contextAlwaysVisible={!isDesktop}
              classes={{ row: classes.row }}
            >
              {isDesktop && <TextField source="id" label={'#'} />}
              <SongTitleField source="title" showTrackNumbers={false} />
              {isDesktop && <AlbumLinkField source="album" />}
              {isDesktop && <TextField source="artist" />}
              <DurationField source="duration" className={classes.draggable} />
              <QualityInfo label="Quality" />
              <SongContextMenu
                onAddToPlaylist={onAddToPlaylist}
                showStar={false}
                className={classes.contextMenu}
              />
            </SongDatagrid>
          </ReorderableList>
        </Card>
      </div>
      <AddToPlaylistDialog />
    </>
  )
}

const SanitizedPlaylistSongs = (props) => {
  const { loaded, loading, total, ...rest } = useListContext(props)
  return (
    <>
      {loaded && (
        <PlaylistSongs
          {...rest}
          playlistId={props.id}
          actions={props.actions}
        />
      )}
    </>
  )
}

export default SanitizedPlaylistSongs
