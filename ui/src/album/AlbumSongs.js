import React from 'react'
import {
  BulkActionsToolbar,
  Datagrid,
  FunctionField,
  ListToolbar,
  TextField,
  useListController
} from 'react-admin'
import classnames from 'classnames'
import { useDispatch } from 'react-redux'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { setTrack } from '../audioplayer'
import { DurationField } from '../common'

const useStyles = makeStyles(
  (theme) => ({
    root: {},
    main: {
      display: 'flex'
    },
    content: {
      marginTop: 0,
      transition: theme.transitions.create('margin-top'),
      position: 'relative',
      flex: '1 1 auto',
      [theme.breakpoints.down('xs')]: {
        boxShadow: 'none'
      },
      overflow: 'inherit'
    },
    bulkActionsDisplayed: {
      marginTop: -theme.spacing(8),
      transition: theme.transitions.create('margin-top')
    },
    actions: {
      zIndex: 2,
      display: 'flex',
      justifyContent: 'flex-end',
      flexWrap: 'wrap'
    },
    noResults: { padding: 20 }
  }),
  { name: 'RaList' }
)

const trackName = (r) => {
  const name = r.title
  if (r.trackNumber) {
    return r.trackNumber.toString().padStart(2, '0') + ' ' + name
  }
  return name
}

const AlbumSongs = (props) => {
  const classes = useStyles(props)
  const dispatch = useDispatch()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const controllerProps = useListController(props)
  const { bulkActionButtons, albumId } = props
  const { loading, data, ids, version } = controllerProps

  if (loading) {
    const anySong = data[ids[0]]
    if (!anySong || anySong.albumId !== albumId) {
      return null
    }
  }

  return (
    <>
      <ListToolbar
        filters={props.filters}
        {...controllerProps}
        actions={props.actions}
        permanentFilter={props.filter}
      />
      <div className={classes.main}>
        <Card
          className={classnames(classes.content, {
            [classes.bulkActionsDisplayed]:
              controllerProps.selectedIds.length > 0
          })}
          key={version}
        >
          {/*{bulkActionButtons !== false && bulkActionButtons && (*/}
          <BulkActionsToolbar {...controllerProps}>
            {bulkActionButtons}
          </BulkActionsToolbar>
          {/*)}*/}
          <Datagrid
            rowClick={(id, basePath, record) => dispatch(setTrack(record))}
            {...controllerProps}
            hasBulkActions={props.bulkActionButtons !== false}
          >
            {isDesktop && (
              <TextField
                source="trackNumber"
                sortBy="discNumber asc, trackNumber asc"
                label="#"
              />
            )}
            {isDesktop && <TextField source="title" />}
            {!isDesktop && <FunctionField source="title" render={trackName} />}
            {isDesktop && <TextField source="artist" />}
            <DurationField source="duration" />
          </Datagrid>
        </Card>
      </div>
    </>
  )
}

export default AlbumSongs
