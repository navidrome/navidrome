import React from 'react'
import { useSelector } from 'react-redux'
import { useGetOne } from 'react-admin'
import PlaylistDetails from './PlaylistDetails'
import { Title } from '../common'
import PlaylistSongs from './PlaylistSongs'
import PlaylistActions from './PlaylistActions'
import PlaylistSongBulkActions from './PlaylistSongBulkActions'
import { isReadOnly } from '../common/Writable'

const PlaylistShow = (props) => {
  const viewVersion = useSelector((s) => s.admin.ui && s.admin.ui.viewVersion)
  const { data: record, error } = useGetOne('playlist', props.id, {
    v: viewVersion,
  })

  if (error) {
    return <p>ERROR: {error}</p>
  }

  return (
    <>
      <PlaylistDetails {...props} record={record} />
      <PlaylistSongs
        {...props}
        playlistId={props.id}
        readOnly={isReadOnly(record && record.owner)}
        title={<Title subTitle={record && record.name} />}
        actions={<PlaylistActions record={record} />}
        filter={{ playlist_id: props.id }}
        resource={'playlistTrack'}
        exporter={false}
        perPage={0}
        pagination={null}
        bulkActionButtons={
          <PlaylistSongBulkActions playlistId={props.id} record={record} />
        }
      />
    </>
  )
}

export default PlaylistShow
