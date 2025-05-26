import React, { useState } from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import Card from '@material-ui/core/Card'
import CardMedia from '@material-ui/core/CardMedia'
import config from '../config'
import { LoveButton, RatingField } from '../common'
import Lightbox from 'react-image-lightbox'
import subsonic from '../subsonic'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
      background: ({ img }) => `url(${img})`,
    },
    bgContainer: {
      display: 'flex',
      height: '15rem',
      width: '100vw',
      padding: 'unset',
      backdropFilter: 'blur(1px)',
      backgroundPosition: '50% 30%',
      background: `linear-gradient(to bottom, rgba(52 52 52 / 72%), rgba(21 21 21))`,
    },
    link: {
      margin: '1px',
    },
    details: {
      display: 'flex',
      alignItems: 'flex-start',
      flexDirection: 'column',
      justifyContent: 'center',
      marginLeft: '0.5rem',
    },
    biography: {
      display: 'flex',
      marginLeft: '3%',
      marginRight: '3%',
      marginTop: '-2em',
      zIndex: '1',
      '& p': {
        whiteSpace: ({ expanded }) => (expanded ? 'unset' : 'nowrap'),
        overflow: 'hidden',
        width: '95vw',
        textOverflow: 'ellipsis',
      },
    },
    cover: {
      width: 151,
      boxShadow: '0px 0px 6px 0px #565656',
      borderRadius: '5px',
      backgroundColor: 'transparent',
      transition: 'opacity 0.3s ease-in-out',
      objectFit: 'cover',
    },
    coverLoading: {
      opacity: 0.5,
    },
    artistImage: {
      marginLeft: '1em',
      maxHeight: '7rem',
      backgroundColor: 'inherit',
      marginTop: '4rem',
      width: '7rem',
      minWidth: '7rem',
      display: 'flex',
      borderRadius: '5em',
    },
    loveButton: {
      top: theme.spacing(-0.2),
      left: theme.spacing(0.5),
    },
    rating: {
      marginTop: '5px',
    },
    artistName: {
      wordBreak: 'break-word',
    },
  }),
  { name: 'NDMobileArtistDetails' },
)

const MobileArtistDetails = ({ artistInfo, biography, record }) => {
  const img = subsonic.getCoverArtUrl(record)
  const [expanded, setExpanded] = useState(false)
  const classes = useStyles({ img, expanded })
  const title = record.name
  const [isLightboxOpen, setLightboxOpen] = React.useState(false)
  const [imageLoading, setImageLoading] = React.useState(false)
  const [imageError, setImageError] = React.useState(false)

  // Reset image state when artist changes
  React.useEffect(() => {
    setImageLoading(true)
    setImageError(false)
  }, [record.id])

  const handleImageLoad = React.useCallback(() => {
    setImageLoading(false)
    setImageError(false)
  }, [])

  const handleImageError = React.useCallback(() => {
    setImageLoading(false)
    setImageError(true)
  }, [])

  const handleOpenLightbox = React.useCallback(() => {
    if (!imageError) {
      setLightboxOpen(true)
    }
  }, [imageError])

  const handleCloseLightbox = React.useCallback(
    () => setLightboxOpen(false),
    [],
  )

  return (
    <>
      <div className={classes.root}>
        <div className={classes.bgContainer}>
          <Card className={classes.artistImage}>
            {artistInfo && (
              <CardMedia
                key={record.id}
                component="img"
                src={subsonic.getCoverArtUrl(record, 300)}
                className={`${classes.cover} ${imageLoading ? classes.coverLoading : ''}`}
                onClick={handleOpenLightbox}
                onLoad={handleImageLoad}
                onError={handleImageError}
                title={title}
                style={{
                  cursor: imageError ? 'default' : 'pointer',
                }}
              />
            )}
          </Card>
          <div className={classes.details}>
            <Typography
              component="h5"
              variant="h5"
              className={classes.artistName}
            >
              {title}
              <LoveButton
                className={classes.loveButton}
                record={record}
                resource={'artist'}
                size={'small'}
                aria-label="love"
                color="primary"
              />
            </Typography>
            {config.enableStarRating && (
              <RatingField
                record={record}
                resource={'artist'}
                size={'small'}
                className={classes.rating}
              />
            )}
          </div>
        </div>
      </div>
      <div className={classes.biography}>
        <Collapse collapsedHeight={'1.5em'} in={expanded} timeout={'auto'}>
          <Typography variant={'body1'} onClick={() => setExpanded(!expanded)}>
            <span dangerouslySetInnerHTML={{ __html: biography }} />
          </Typography>
        </Collapse>
      </div>
      {isLightboxOpen && !imageError && (
        <Lightbox
          imagePadding={50}
          animationDuration={200}
          imageTitle={record.name}
          mainSrc={img}
          onCloseRequest={handleCloseLightbox}
        />
      )}
    </>
  )
}

export default MobileArtistDetails
