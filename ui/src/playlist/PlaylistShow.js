import React from 'react'
import { useGetOne } from 'react-admin'
import PlaylistDetails from './PlaylistDetails'
import { Title } from '../common'
import PlaylistSongs from './PlaylistSongs'

const PlaylistShow = (props) => {
  const { data: record, loading, error } = useGetOne('playlist', props.id)

  if (loading) {
    return null
  }

  if (error) {
    return <p>ERROR: {error}</p>
  }

  return (
    <>
      <PlaylistDetails {...props} record={record} />
      <PlaylistSongs
        {...props}
        playlistId={props.id}
        title={<Title subTitle={record.name} />}
        // actions={<AlbumActions />}
        filter={{ playlist_id: props.id }}
        resource={'playlistTrack'}
        exporter={false}
        perPage={-1}
        pagination={null}
        bulkActionButtons={false}
        // bulkActionButtons={<AlbumSongBulkActions />}
      />
    </>
  )
}

export default PlaylistShow
