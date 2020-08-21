import React from 'react'
import { Card, CardContent, CardMedia, Typography } from '@material-ui/core'
import { useTranslate } from 'react-admin'
import Lightbox from 'react-image-lightbox'
import 'react-image-lightbox/style.css'
import subsonic from '../subsonic'
import { DurationField, formatRange } from '../common'
import { ArtistLinkField } from '../common'

const AlbumDetails = ({ classes, record }) => {
  const [isLightboxOpen, setLightboxOpen] = React.useState(false)
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

  const imageUrl = subsonic.url('getCoverArt', record.coverArtId || 'not_found')

  const handleOpenLightbox = React.useCallback(() => setLightboxOpen(true), [])
  const handleCloseLightbox = React.useCallback(
    () => setLightboxOpen(false),
    []
  )

  return (
    <Card className={classes.container}>
      <CardMedia
        image={imageUrl}
        className={classes.albumCover}
        onClick={handleOpenLightbox}
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

      {isLightboxOpen && (
        <Lightbox
          imagePadding={50}
          animationDuration={200}
          imageTitle={record.name}
          mainSrc={imageUrl}
          onCloseRequest={handleCloseLightbox}
        />
      )}
    </Card>
  )
}

export default AlbumDetails
