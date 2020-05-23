import React from 'react'
import {
  BulkActionsToolbar,
  DatagridLoading,
  ListToolbar,
  TextField,
  DatagridBody,
  Datagrid,
  useListController,
  useRefresh,
} from 'react-admin'
import classnames from 'classnames'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import {
  DurationField,
  SongDetails,
  SongContextMenu,
  SongDatagridRow,
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

const SongsDatagridBody = (props) => (
  <DatagridBody {...props} row={<SongDatagridRow contextVisible={true} />} />
)

const SongsDatagrid = ({ contextVisible, ...rest }) => {
  return <Datagrid {...rest} body={<SongsDatagridBody />} />
}

const PlaylistSongs = (props) => {
  const classes = useStyles(props)
  const classesToolbar = useStylesListToolbar(props)
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const controllerProps = useListController(props)
  const refresh = useRefresh()
  const { bulkActionButtons, expand, className, playlistId } = props
  const { data, ids, version, loaded } = controllerProps

  const anySong = data[ids[0]]
  const showPlaceholder = !anySong || anySong.playlistId !== playlistId
  const hasBulkActions = props.bulkActionButtons !== false

  if (loaded && ids.length === 0) {
    return <div />
  }

  const onAddToPlaylist = (playlistId) => {
    if (playlistId === props.id) {
      refresh()
    }
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
            <SongsDatagrid
              expand={!isXsmall && <SongDetails />}
              rowClick={null}
              {...controllerProps}
              hasBulkActions={hasBulkActions}
            >
              {isDesktop && <TextField source="id" label={'#'} />}
              <TextField source="title" />
              {isDesktop && <TextField source="artist" />}
              <DurationField source="duration" />
              <SongContextMenu
                onAddToPlaylist={onAddToPlaylist}
                showStar={false}
              />
            </SongsDatagrid>
          )}
        </Card>
      </div>
    </>
  )
}

export default PlaylistSongs
