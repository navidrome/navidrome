import React from 'react'
import {
  Datagrid,
  FunctionField,
  List,
  Loading,
  TextField,
  useGetOne
} from 'react-admin'
import AlbumDetails from './AlbumDetails'
import { DurationField, Title } from '../common'
import { useStyles } from './styles'
import { AlbumActions } from './AlbumActions'
import { AlbumSongBulkActions } from './AlbumSongBulkActions'
import { useMediaQuery } from '@material-ui/core'
import { setTrack } from '../audioplayer'
import { useDispatch } from 'react-redux'

const AlbumShow = (props) => {
  const dispatch = useDispatch()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles()
  const { data: record, loading, error } = useGetOne('album', props.id)

  if (loading) {
    return <Loading />
  }

  if (error) {
    return <p>ERROR: {error}</p>
  }

  const trackName = (r) => {
    const name = r.title
    if (r.trackNumber) {
      return r.trackNumber.toString().padStart(2, '0') + ' ' + name
    }
    return name
  }

  return (
    <>
      <AlbumDetails {...props} classes={classes} record={record} />
      <List
        {...props}
        title={<Title subTitle={record.name} />}
        actions={<AlbumActions />}
        filter={{ album_id: props.id }}
        resource={'albumSong'}
        exporter={false}
        perPage={1000}
        pagination={null}
        sort={{ field: 'discNumber asc, trackNumber asc', order: 'ASC' }}
        bulkActionButtons={<AlbumSongBulkActions />}
      >
        <Datagrid
          rowClick={(id, basePath, record) => dispatch(setTrack(record))}
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
          {record.compilation && <TextField source="artist" />}
          <DurationField source="duration" />
        </Datagrid>
      </List>
    </>
  )
}

export default AlbumShow
