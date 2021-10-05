import React from 'react'

import { Typography, Collapse, Link } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import Card from '@material-ui/core/Card'
import CardMedia from '@material-ui/core/CardMedia'
import { useTranslate } from 'react-admin'

const useStyles = makeStyles(
  () => ({
    root: {
      display: 'flex',
      padding: '1em',
      '& .MuiTypography-h5': {
        wordBreak: 'break-word',
      },
      padding: 'unset',
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
    mdetails: {
      display: 'flex',
      alignItems: 'center',
      width: '7rem',
      marginLeft: '0.5rem',
      flex: '1',
    },
    mbio: {
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
    martImage: {
      marginLeft: '1em',
      maxHeight: '10rem',
      backgroundColor: 'inherit',
      marginTop: '4rem',
      maxHeight: '7rem',
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
  completeBioLink,
  handleExpandClick,
  ...props
}) => {
  const translate = useTranslate()
  const classes = useStyles({ img, expanded })
  console.log('props are', artistInfo)

  return (
    <>
      <div className={classes.root}>
        <div className={classes.bgContainer}>
          <Card className={classes.martImage}>
            {artistInfo && (
              <CardMedia
                className={classes.cover}
                image={`${artistInfo.mediumImageUrl}`}
                title={title}
              />
            )}
          </Card>
          <div className={classes.mdetails}>
            <Typography component="h5" variant="h5">
              {title}
            </Typography>
          </div>
          {/* <Card className={classes.artDetail}>
                <Card className={classes.artImage}>
                {artistInfo && (
                    <CardMedia
                    className={classes.cover}
                    image={`${artistInfo.mediumImageUrl}`}
                    title={title}
                    />
                )}
                </Card>
            </Card> */}
        </div>
      </div>
      <div className={classes.mbio}>
        <Collapse collapsedHeight={'1.5em'} in={expanded} timeout={'auto'}>
          <Typography variant={'body1'} onClick={handleExpandClick}>
            {biography}
            <Link
              href={completeBioLink}
              className={classes.link}
              target="_blank"
              rel="nofollow"
            >
              {translate('message.lastfmLink')}
            </Link>
          </Typography>
        </Collapse>
      </div>
    </>
  )
}

export default MartistDetails
