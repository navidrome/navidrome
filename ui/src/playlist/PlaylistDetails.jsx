import {
  Card,
  CardContent,
  CardMedia,
  Typography,
  useMediaQuery,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'
import { useCallback, useState, useEffect } from 'react'
import Lightbox from 'react-image-lightbox'
import 'react-image-lightbox/style.css'
import { CollapsibleComment, DurationField, SizeField } from '../common'
import subsonic from '../subsonic'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      [theme.breakpoints.down('xs')]: {
        padding: '0.7em',
        minWidth: '20em',
      },
      [theme.breakpoints.up('sm')]: {
        padding: '1em',
        minWidth: '32em',
      },
    },
    cardContents: {
      display: 'flex',
    },
    details: {
      display: 'flex',
      flexDirection: 'column',
    },
    content: {
      flex: '2 0 auto',
    },
    coverParent: {
      [theme.breakpoints.down('xs')]: {
        height: '8em',
        width: '8em',
        minWidth: '8em',
      },
      [theme.breakpoints.up('sm')]: {
        height: '10em',
        width: '10em',
        minWidth: '10em',
      },
      [theme.breakpoints.up('lg')]: {
        height: '15em',
        width: '15em',
        minWidth: '15em',
      },
      backgroundColor: 'transparent',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    },
    cover: {
      objectFit: 'contain',
      cursor: 'pointer',
      display: 'block',
      width: '100%',
      height: '100%',
      backgroundColor: 'transparent',
      transition: 'opacity 0.3s ease-in-out',
    },
    coverLoading: {
      opacity: 0.5,
    },
    title: {
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      wordBreak: 'break-word',
    },
    stats: {
      marginTop: '1em',
      marginBottom: '0.5em',
    },
  }),
  {
    name: 'NDPlaylistDetails',
  },
)

const PlaylistDetails = (props) => {
  const { record = {} } = props
  const translate = useTranslate()
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('lg'))
  const [isLightboxOpen, setLightboxOpen] = useState(false)
  const [imageLoading, setImageLoading] = useState(false)
  const [imageError, setImageError] = useState(false)

  const imageUrl = subsonic.getCoverArtUrl(record, 300, true)
  const fullImageUrl = subsonic.getCoverArtUrl(record)

  // Reset image state when playlist changes
  useEffect(() => {
    setImageLoading(true)
    setImageError(false)
  }, [record.id])

  const handleImageLoad = useCallback(() => {
    setImageLoading(false)
    setImageError(false)
  }, [])

  const handleImageError = useCallback(() => {
    setImageLoading(false)
    setImageError(true)
  }, [])

  const handleOpenLightbox = useCallback(() => {
    if (!imageError) {
      setLightboxOpen(true)
    }
  }, [imageError])

  const handleCloseLightbox = useCallback(() => setLightboxOpen(false), [])

  return (
    <Card className={classes.root}>
      <div className={classes.cardContents}>
        <div className={classes.coverParent}>
          <CardMedia
            key={record.id} // Force re-render when playlist changes
            component={'img'}
            src={imageUrl}
            width="400"
            height="400"
            className={`${classes.cover} ${imageLoading ? classes.coverLoading : ''}`}
            onClick={handleOpenLightbox}
            onLoad={handleImageLoad}
            onError={handleImageError}
            title={record.name}
            style={{
              cursor: imageError ? 'default' : 'pointer',
            }}
          />
        </div>
        <div className={classes.details}>
          <CardContent className={classes.content}>
            <Typography
              variant={isDesktop ? 'h5' : 'h6'}
              className={classes.title}
            >
              {record.name || translate('ra.page.loading')}
            </Typography>
            <Typography component="p" className={classes.stats}>
              {record.songCount ? (
                <span>
                  {record.songCount}{' '}
                  {translate('resources.song.name', {
                    smart_count: record.songCount,
                  })}
                  {' · '}
                  <DurationField record={record} source={'duration'} />
                  {' · '}
                  <SizeField record={record} source={'size'} />
                </span>
              ) : (
                <span>&nbsp;</span>
              )}
            </Typography>
            <CollapsibleComment record={record} />
          </CardContent>
        </div>
      </div>
      {isLightboxOpen && !imageError && (
        <Lightbox
          imagePadding={50}
          animationDuration={200}
          imageTitle={record.name}
          mainSrc={fullImageUrl}
          onCloseRequest={handleCloseLightbox}
        />
      )}
    </Card>
  )
}

export default PlaylistDetails
