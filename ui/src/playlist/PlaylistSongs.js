import React from 'react'
import {
  BulkActionsToolbar,
  DatagridLoading,
  ListToolbar,
  TextField,
  useListController,
  useRefresh,
  useDataProvider,
  useNotify,
} from 'react-admin'
import classnames from 'classnames'
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
import AddToPlaylistDialog from '../dialogs/AddToPlaylistDialog'
import { AlbumLinkField } from '../song/AlbumLinkField'
import { playTracks } from '../audioplayer'

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
  }),
  { name: 'RaList' }
)

const useStylesListToolbar = makeStyles({
  toolbar: {
    justifyContent: 'flex-start',
  },
})

const ReorderableList = ({ readOnly, children, ...rest }) => {
  if (readOnly) {
    return children
  }
  return <ReactDragListView {...rest}>{children}</ReactDragListView>
}

const PlaylistSongs = (props) => {
  const classes = useStyles(props)
  const classesToolbar = useStylesListToolbar(props)
  const dispatch = useDispatch()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const controllerProps = useListController(props)
  const dataProvider = useDataProvider()
  const refresh = useRefresh()
  const notify = useNotify()
  const { bulkActionButtons, expand, className, playlistId, readOnly } = props
  const { data, ids, version, total } = controllerProps

  if (total === 0) {
    return null
  }

  const anySong = data[ids[0]]
  const showPlaceholder = !anySong || anySong.playlistId !== playlistId
  const hasBulkActions = props.bulkActionButtons !== false

  const reorder = (playlistId, id, newPos) => {
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
  }

  const onAddToPlaylist = (pls) => {
    if (pls.id === props.id) {
      refresh()
    }
  }

  const handleDragEnd = (from, to) => {
    const toId = ids[to]
    const fromId = ids[from]
    reorder(playlistId, fromId, toId)
  }

  return (
    <>
      <ListToolbar
        classes={classesToolbar}
        filters={props.filters}
        {...controllerProps}
        actions={props.actions}
        permanentFilter={props.filter}
      />
      <div className={classes.main}>
        <Card
          className={classnames(classes.content, {
            [classes.bulkActionsDisplayed]:
              controllerProps.selectedIds.length > 0,
          })}
          key={version}
        >
          {bulkActionButtons !== false && bulkActionButtons && (
            <BulkActionsToolbar {...controllerProps}>
              {bulkActionButtons}
            </BulkActionsToolbar>
          )}
          {showPlaceholder ? (
            <DatagridLoading
              classes={classes}
              className={className}
              expand={expand}
              hasBulkActions={hasBulkActions}
              nbChildren={3}
              size={'small'}
            />
          ) : (
            <ReorderableList
              readOnly={readOnly}
              onDragEnd={handleDragEnd}
              nodeSelector={'tr'}
            >
              <SongDatagrid
                expand={!isXsmall && <SongDetails />}
                rowClick={(id) => dispatch(playTracks(data, ids, id))}
                {...controllerProps}
                hasBulkActions={hasBulkActions}
                contextAlwaysVisible={!isDesktop}
              >
                {isDesktop && <TextField source="id" label={'#'} />}
                <SongTitleField source="title" showTrackNumbers={false} />
                {isDesktop && <AlbumLinkField source="album" />}
                {isDesktop && <TextField source="artist" />}
                <DurationField
                  source="duration"
                  className={classes.draggable}
                />
                <SongContextMenu
                  onAddToPlaylist={onAddToPlaylist}
                  showStar={false}
                />
              </SongDatagrid>
            </ReorderableList>
          )}
        </Card>
      </div>
      <AddToPlaylistDialog />
    </>
  )
}

export default PlaylistSongs
