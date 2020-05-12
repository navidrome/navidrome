import React from 'react'
import {
  BulkActionsToolbar,
  Datagrid,
  FunctionField,
  ListToolbar,
  TextField,
  useListController,
  DatagridLoading,
  DatagridBody,
  DatagridRow,
} from 'react-admin'
import classnames from 'classnames'
import { useDispatch } from 'react-redux'
import {
  Card,
  useMediaQuery,
  TableRow,
  TableCell,
  Typography,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { playAlbum } from '../audioplayer'
import { DurationField } from '../common'
import { SongDetails } from '../common'

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
  const { bulkActionButtons, albumId, expand, className } = props
  const { data, ids, version, loaded } = controllerProps

  let multiDisc = false
  if (loaded) {
    const discSet = new Set()
    ids.forEach((id) => discSet.add(data[id].discNumber))
    multiDisc = discSet.size > 1
    console.log(multiDisc, discSet.size)
  }

  const anySong = data[ids[0]]
  const showPlaceholder = !anySong || anySong.albumId !== albumId
  const hasBulkActions = props.bulkActionButtons !== false

  const SongDatagridRow = (props) => {
    const { record, children } = props
    return (
      <>
        {multiDisc && (
          <TableRow>
            {record.trackNumber === 1 && (
              <TableCell colSpan={children.length + 1}>
                <Typography variant="h6">
                  {record.discSubtitle
                    ? `${record.discSubtitle} (disc ${record.discNumber})`
                    : `Disc ${record.discNumber}`}
                </Typography>
              </TableCell>
            )}
          </TableRow>
        )}
        <DatagridRow {...props} />
      </>
    )
  }

  const SongsDatagridBody = (props) => (
    <DatagridBody {...props} row={<SongDatagridRow />} />
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
              expand={expand}
              hasBulkActions={hasBulkActions}
              nbChildren={3}
              size={'small'}
            />
          ) : (
            <SongsDatagrid
              expand={!isXsmall && <SongDetails />}
              rowClick={(id) => dispatch(playAlbum(data, ids, id))}
              {...controllerProps}
              hasBulkActions={hasBulkActions}
              multiDisc={multiDisc}
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
            </SongsDatagrid>
          )}
        </Card>
      </div>
    </>
  )
}

export default AlbumSongs
