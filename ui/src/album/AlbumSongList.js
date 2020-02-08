import React from 'react'
import { useGetList } from 'react-admin'
import { DurationField, PlayButton, SimpleList } from '../common'
import { addTrack } from '../player'
import AddIcon from '@material-ui/icons/Add'
import { useDispatch } from 'react-redux'
import { playAlbum } from '../player/queue'

const AlbumSongList = (props) => {
  const dispatch = useDispatch()
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
      return r.trackNumber.toString().padStart(2, '0') + ' ' + name
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
          <PlayButton action={playAlbum(r.id, data)} />
          <PlayButton action={addTrack(r)} icon={<AddIcon />} />
          {trackName(r)}
        </>
      )}
      secondaryText={(r) =>
        r.albumArtist && r.artist !== r.albumArtist ? r.artist : ''
      }
      tertiaryText={(r) => <DurationField record={r} source={'duration'} />}
      linkType={(id) => dispatch(playAlbum(id, data))}
    />
  )
}

export default AlbumSongList
