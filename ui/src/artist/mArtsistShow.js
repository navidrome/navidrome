import React from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import Card from '@material-ui/core/Card'
import CardMedia from '@material-ui/core/CardMedia'
import ArtistExternalLinks from './ArtistExternalLink'

const useStyles = makeStyles(
  () => ({
    root: {
      display: 'flex',
      '& .MuiTypography-h5': {
        wordBreak: 'break-word',
      },
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
      alignItems: 'center',
      width: '7rem',
      marginLeft: '0.5rem',
      flex: '1',
    },
    bio: {
      display: 'flex',
      marginLeft: '3%',
      marginRight: '3%',
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
      display: 'flex',
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
            <Typography component="h5" variant="h5">
              {title}
            </Typography>
          </div>
        </div>
      </div>
      <div className={classes.bio}>
        <Collapse collapsedHeight={'1.5em'} in={expanded} timeout={'auto'}>
          <Typography variant={'body1'} onClick={handleExpandClick}>
            {biography}
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
