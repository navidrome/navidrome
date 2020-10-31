import React from 'react'
import { useGetOne } from 'react-admin'
import AlbumDetails from './AlbumDetails'
import { Title } from '../common'
import { useStyles } from './styles'
import { SongBulkActions } from '../common'
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
        actions={<AlbumActions record={record} />}
        filter={{ album_id: props.id }}
        resource={'albumSong'}
        exporter={false}
        perPage={0}
        pagination={null}
        sort={{ field: 'discNumber asc, trackNumber asc', order: 'ASC' }}
        bulkActionButtons={<SongBulkActions />}
      />
    </>
  )
}

export default AlbumShow
