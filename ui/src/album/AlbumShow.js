import React from 'react'
import { useGetOne } from 'react-admin'
import AlbumDetails from './AlbumDetails'
import { Title } from '../common'
import { useStyles } from './styles'
import { AlbumSongBulkActions } from './AlbumSongBulkActions'
import AlbumActions from './AlbumActions'
import AlbumSongs from './AlbumSongs'

const AlbumShow = (props) => {
  const classes = useStyles()
  const { data: record, loading, error } = useGetOne('album', props.id)

  if (loading) {
    return null
  }

  if (error) {
    return <p>ERROR: {error}</p>
  }

  return (
    <>
      <AlbumDetails {...props} classes={classes} record={record} />
      <AlbumSongs
        {...props}
        albumId={props.id}
        title={<Title subTitle={record.name} />}
        actions={<AlbumActions albumId={props.id} />}
        filter={{ album_id: props.id }}
        resource={'albumSong'}
        exporter={false}
        perPage={-1}
        pagination={null}
        sort={{ field: 'discNumber asc, trackNumber asc', order: 'ASC' }}
        bulkActionButtons={<AlbumSongBulkActions />}
      />
    </>
  )
}

export default AlbumShow
