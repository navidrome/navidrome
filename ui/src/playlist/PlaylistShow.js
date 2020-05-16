import React from 'react'
import { useSelector } from 'react-redux'
import { useGetOne } from 'react-admin'
import PlaylistDetails from './PlaylistDetails'
import { Title } from '../common'
import PlaylistSongs from './PlaylistSongs'
import PlaylistActions from './PlaylistActions'
import PlaylistSongBulkActions from './PlaylistSongBulkActions'

const PlaylistShow = (props) => {
  const viewVersion = useSelector((s) => s.admin.ui && s.admin.ui.viewVersion)
  const { data: record, loading, error } = useGetOne('playlist', props.id, {
    v: viewVersion,
  })

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
        actions={<PlaylistActions />}
        filter={{ playlist_id: props.id }}
        resource={'playlistTrack'}
        exporter={false}
        perPage={-1}
        pagination={null}
        bulkActionButtons={<PlaylistSongBulkActions playlistId={props.id} />}
      />
    </>
  )
}

export default PlaylistShow
