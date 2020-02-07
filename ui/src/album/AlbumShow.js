import React from 'react'
import { Show, SimpleList, useGetList, useGetOne, Loading } from 'react-admin'
import { PlayButton, Title } from '../common'
import { addTrack } from '../player'
import { DurationField } from '../common'
import AddIcon from '@material-ui/icons/Add'
import { Typography, Paper } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'

const AlbumTitle = ({ record }) => {
  return <Title subTitle={`Album: ${record ? record.name : ''}`} />
}

const useStyles = makeStyles({
  container: { minWidth: '35em', padding: '1em' },
  rightAlignedCell: { textAlign: 'right' },
  boldCell: { fontWeight: 'bold' }
})

const AlbumDetail = (props) => {
  const classes = useStyles()
  const { data, loading, error } = useGetOne('album', props.id)

  if (loading) {
    return <Loading />
  }

  if (error) {
    return <p>ERROR: {error}</p>
  }

  let genreDate = []
  if (data.genre) {
    genreDate.push(data.genre)
  }
  if (data.year) {
    genreDate.push(data.year)
  }

  return (
    <Paper className={classes.container} elevation={2}>
      <Typography variant="h5">{data.name}</Typography>
      <Typography variant="h6">{data.albumArtist || data.artist}</Typography>
      <Typography variant="h7">{genreDate.join(' - ')}</Typography>
      <Typography variant="body2"></Typography>
    </Paper>
  )
}

const AlbumSongs = (props) => {
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
          {r.trackNumber + '. ' + r.title}
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

const AlbumShow = (props) => {
  return (
    <>
      <AlbumDetail {...props} />
      <Show title={<AlbumTitle />} {...props}>
        <AlbumSongs {...props} />
      </Show>
    </>
  )
}

export default AlbumShow
