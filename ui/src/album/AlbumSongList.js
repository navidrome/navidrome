import React from 'react'
import { SimpleList, useGetList } from 'react-admin'
import { DurationField, PlayButton } from '../common'
import { addTrack } from '../player'
import AddIcon from '@material-ui/icons/Add'

const AlbumSongList = (props) => {
  const { record } = props
  const { data, total, loading, error } = useGetList(
    'song',
    { page: 0, perPage: 100 },
    { field: 'album', order: 'ASC' },
    { album_id: record.id }
  )
  if (error) {
    return <p>ERROR: {error}</p>
  }
  const trackName = (r) => {
    const name = r.title
    if (r.trackNumber) {
      return r.trackNumber + '. ' + name
    }
    return name
  }
  return (
    <SimpleList
      data={data}
      ids={Object.keys(data)}
      loading={loading}
      total={total}
      primaryText={(r) => (
        <>
          <PlayButton record={r} />
          <PlayButton record={r} action={addTrack} icon={<AddIcon />} />
          {trackName(r)}
        </>
      )}
      secondaryText={(r) =>
        r.albumArtist && r.artist !== r.albumArtist ? r.artist : ''
      }
      tertiaryText={(r) => <DurationField record={r} source={'duration'} />}
      linkType={false}
    />
  )
}

export default AlbumSongList
