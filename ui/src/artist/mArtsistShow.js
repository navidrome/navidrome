import React from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import Card from '@material-ui/core/Card'
import CardMedia from '@material-ui/core/CardMedia'
import ArtistExternalLinks from './ArtistExternalLink'
import config from '../config'
import { LoveButton, RatingField } from '../common'
import { useTranslate } from 'ra-core'

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
    bio: {
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
    },
    artImage: {
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
    artName: {
      wordBreak: 'break-word',
    },
  }),
  { name: 'mNDArtistPage' }
)

const MartistDetails = ({
  img,
  expanded,
  artistInfo,
  title,
  biography,
  record,
  handleExpandClick,
}) => {
  const classes = useStyles({ img, expanded })
  const translate = useTranslate()

  return (
    <>
      <div className={classes.root}>
        <div className={classes.bgContainer}>
          <Card className={classes.artImage}>
            {artistInfo && (
              <CardMedia
                className={classes.cover}
                image={`${artistInfo.mediumImageUrl}`}
                title={title}
              />
            )}
          </Card>
          <div className={classes.details}>
            <Typography component="h5" variant="h5" className={classes.artName}>
              {title}
              {config.enableFavourites && (
                <LoveButton
                  className={classes.loveButton}
                  record={record}
                  resource={'artist'}
                  size={'small'}
                  aria-label="love"
                  color="primary"
                />
              )}
            </Typography>
            {record.albumCount +
              ' ' +
              translate('resources.album.name', {
                smart_count: record.albumCount,
              })}
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
      <div className={classes.bio}>
        <Collapse collapsedHeight={'1.5em'} in={expanded} timeout={'auto'}>
          <Typography variant={'body1'} onClick={handleExpandClick}>
            <span dangerouslySetInnerHTML={{ __html: biography }} />
            <ArtistExternalLinks
              className={classes.link}
              record={record}
              artistInfo={artistInfo}
            />
          </Typography>
        </Collapse>
      </div>
    </>
  )
}

export default MartistDetails
