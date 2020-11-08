import React from 'react'
import {
  BulkActionsToolbar,
  DatagridLoading,
  ListToolbar,
  TextField,
  useListController,
} from 'react-admin'
import classnames from 'classnames'
import { useDispatch } from 'react-redux'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import { playTracks } from '../actions'
import {
  DurationField,
  SongContextMenu,
  SongDatagrid,
  SongDetails,
  SongTitleField,
} from '../common'
import AddToPlaylistDialog from '../dialogs/AddToPlaylistDialog'

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
  }),
  { name: 'RaList' }
)

const useStylesListToolbar = makeStyles({
  toolbar: {
    justifyContent: 'flex-start',
  },
})

const AlbumSongs = (props) => {
  const classes = useStyles(props)
  const classesToolbar = useStylesListToolbar(props)
  const dispatch = useDispatch()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const controllerProps = useListController(props)
  const { bulkActionButtons, albumId, className } = props
  const { data, ids, version } = controllerProps

  const anySong = data[ids[0]]
  const showPlaceholder = !anySong || anySong.albumId !== albumId
  const hasBulkActions = props.bulkActionButtons !== false

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
            <SongDatagrid
              expand={isXsmall ? null : <SongDetails />}
              rowClick={(id) => dispatch(playTracks(data, ids, id))}
              {...controllerProps}
              hasBulkActions={hasBulkActions}
              showDiscSubtitles={true}
              contextAlwaysVisible={!isDesktop}
            >
              {isDesktop && (
                <TextField
                  source="trackNumber"
                  sortBy="discNumber asc, trackNumber asc"
                  label="#"
                  sortable={false}
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
                label={
                  <StarBorderIcon
                    fontSize={'small'}
                    className={classes.columnIcon}
                  />
                }
              />
            </SongDatagrid>
          )}
        </Card>
      </div>
      <AddToPlaylistDialog />
    </>
  )
}

export default AlbumSongs
