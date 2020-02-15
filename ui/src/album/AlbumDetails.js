import React from 'react'
import { Card, CardContent, CardMedia, Typography } from '@material-ui/core'
import { useTranslate } from 'react-admin'
import { subsonicUrl } from '../subsonic'
import { DurationField } from '../common'

const AlbumDetails = ({ classes, record }) => {
  const translate = useTranslate()
  const genreYear = (record) => {
    let genreDateLine = []
    if (record.genre) {
      genreDateLine.push(record.genre)
    }
    if (record.year) {
      genreDateLine.push(record.year)
    }
    return genreDateLine.join(' · ')
  }

  return (
    <Card className={classes.container}>
      <CardMedia
        image={subsonicUrl('getCoverArt', record.coverArtId || 'not_found', {
          size: 500
        })}
        className={classes.albumCover}
      />
      <CardContent className={classes.albumDetails}>
        <Typography variant="h5" className={classes.albumTitle}>
          {record.name}
        </Typography>
        <Typography component="h6">
          {record.albumArtist || record.artist}
        </Typography>
        <Typography component="p">{genreYear(record)}</Typography>
        <Typography component="p">
          {record.songCount}{' '}
          {translate('resources.song.name', { smart_count: record.songCount })}{' '}
          · <DurationField record={record} source={'duration'} />
        </Typography>
      </CardContent>
    </Card>
  )
}

export default AlbumDetails
