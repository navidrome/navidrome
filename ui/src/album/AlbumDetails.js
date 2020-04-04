import React from 'react'
import { Card, CardContent, CardMedia, Typography } from '@material-ui/core'
import { useTranslate } from 'react-admin'
import subsonic from '../subsonic'
import { DurationField, formatRange } from '../common'
import { ArtistLinkField } from './ArtistLinkField'

const AlbumDetails = ({ classes, record }) => {
  const translate = useTranslate()
  const genreYear = (record) => {
    let genreDateLine = []
    if (record.genre) {
      genreDateLine.push(record.genre)
    }
    const year = formatRange(record, 'year')
    if (year) {
      genreDateLine.push(year)
    }
    return genreDateLine.join(' · ')
  }

  return (
    <Card className={classes.container}>
      <CardMedia
        image={subsonic.url('getCoverArt', record.coverArtId || 'not_found')}
        className={classes.albumCover}
      />
      <CardContent className={classes.albumDetails}>
        <Typography variant="h5" className={classes.albumTitle}>
          {record.name}
        </Typography>
        <Typography component="h6">
          <ArtistLinkField record={record} />
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
