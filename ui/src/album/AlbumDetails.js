import React from 'react'
import { Loading, useGetOne } from 'react-admin'
import { Card, CardContent, CardMedia, Typography } from '@material-ui/core'
import { subsonicUrl } from '../subsonic'

const AlbumDetails = ({ id, classes }) => {
  const { data, loading, error } = useGetOne('album', id)

  if (loading) {
    return <Loading />
  }

  if (error) {
    return <p>ERROR: {error}</p>
  }

  const genreYear = (data) => {
    let genreDateLine = []
    if (data.genre) {
      genreDateLine.push(data.genre)
    }
    if (data.year) {
      genreDateLine.push(data.year)
    }
    return genreDateLine.join(' - ')
  }

  return (
    <Card className={classes.container}>
      <CardMedia
        image={subsonicUrl(
          'getCoverArt',
          data.coverArtId || 'not_found',
          'size=500'
        )}
        className={classes.albumCover}
      />
      <CardContent className={classes.albumDetails}>
        <Typography variant="h5" className={classes.albumTitle}>
          {data.name}
        </Typography>
        <Typography component="h6">
          {data.albumArtist || data.artist}
        </Typography>
        <Typography component="p">{genreYear(data)}</Typography>
      </CardContent>
    </Card>
  )
}

export default AlbumDetails
