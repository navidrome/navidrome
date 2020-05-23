import React from 'react'
import {
  BulkActionsToolbar,
  Datagrid,
  DatagridBody,
  DatagridLoading,
  FunctionField,
  ListToolbar,
  TextField,
  useListController,
} from 'react-admin'
import classnames from 'classnames'
import { useDispatch } from 'react-redux'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { playTracks } from '../audioplayer'
import {
  DurationField,
  SongDetails,
  SongDatagridRow,
  SongContextMenu,
} from '../common'

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

const trackName = (r) => {
  const name = r.title
  if (r.trackNumber) {
    return r.trackNumber.toString().padStart(2, '0') + ' ' + name
  }
  return name
}

const AlbumSongs = (props) => {
  const classes = useStyles(props)
  const classesToolbar = useStylesListToolbar(props)
  const dispatch = useDispatch()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const controllerProps = useListController(props)
  const { bulkActionButtons, albumId, className } = props
  const { data, ids, version, loaded } = controllerProps

  let multiDisc = false
  if (loaded) {
    const discNumbers = ids
      .map((id) => data[id])
      .filter((r) => r)
      .map((r) => r.discNumber)
    multiDisc = new Set(discNumbers).size > 1
  }

  const anySong = data[ids[0]]
  const showPlaceholder = !anySong || anySong.albumId !== albumId
  const hasBulkActions = props.bulkActionButtons !== false

  const SongsDatagridBody = (props) => (
    <DatagridBody {...props} row={<SongDatagridRow multiDisc={multiDisc} />} />
  )
  const SongsDatagrid = (props) => (
    <Datagrid {...props} body={<SongsDatagridBody />} />
  )

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
              expand={null}
              hasBulkActions={hasBulkActions}
              nbChildren={3}
              size={'small'}
            />
          ) : (
            <SongsDatagrid
              expand={!isXsmall && <SongDetails />}
              rowClick={(id) => dispatch(playTracks(data, ids, id))}
              {...controllerProps}
              hasBulkActions={hasBulkActions}
            >
              {isDesktop && (
                <TextField
                  source="trackNumber"
                  sortBy="discNumber asc, trackNumber asc"
                  label="#"
                  sortable={false}
                />
              )}
              {isDesktop && <TextField source="title" sortable={false} />}
              {!isDesktop && (
                <FunctionField
                  source="title"
                  render={trackName}
                  sortable={false}
                />
              )}
              {isDesktop && <TextField source="artist" sortable={false} />}
              <DurationField source="duration" sortable={false} />
              <SongContextMenu />
            </SongsDatagrid>
          )}
        </Card>
      </div>
    </>
  )
}

export default AlbumSongs
