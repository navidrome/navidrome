import React from 'react'
import {
  GridList,
  GridListTile,
  Typography,
  GridListTileBar,
  useMediaQuery,
} from '@material-ui/core'
import { makeStyles, useTheme } from '@material-ui/core/styles'
import withWidth from '@material-ui/core/withWidth'
import { Link } from 'react-router-dom'
import { linkToRecord, useListContext, Loading } from 'react-admin'
import { withContentRect } from 'react-measure'
import subsonic from '../subsonic'

import Card from '@material-ui/core/Card'
import CardContent from '@material-ui/core/CardContent'
import CardMedia from '@material-ui/core/CardMedia'
import IconButton from '@material-ui/core/IconButton'
import SkipPreviousIcon from '@material-ui/icons/SkipPrevious'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import SkipNextIcon from '@material-ui/icons/SkipNext'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
    },
    details: {
      display: 'flex',
      flexDirection: 'column',
    },
    content: {
      flex: '1 0 auto',
    },
    cover: {
      width: 151,
    },
    controls: {
      display: 'flex',
      alignItems: 'center',
      paddingLeft: theme.spacing(1),
      paddingBottom: theme.spacing(1),
    },
    playIcon: {
      height: 38,
      width: 38,
    },
  }),
  { name: 'NDArtistPage' }
)

function ImgMediaCard() {
  const classes = useStyles()
  const theme = useTheme()

  return (
    <Card className={classes.root}>
      <CardMedia
        className={classes.cover}
        image="https://images.unsplash.com/photo-1494548162494-384bba4ab999?ixlib=rb-1.2.1&w=1000&q=80"
        title="Live from space album cover"
      />
      <div className={classes.details}>
        <CardContent className={classes.content}>
          <Typography component="h5" variant="h5">
            Live From Space
          </Typography>
          <Typography variant="subtitle1" color="textSecondary">
            Mac Miller
          </Typography>
        </CardContent>
        <div className={classes.controls}>
          <IconButton aria-label="previous">
            {theme.direction === 'rtl' ? (
              <SkipNextIcon />
            ) : (
              <SkipPreviousIcon />
            )}
          </IconButton>
          <IconButton aria-label="play/pause">
            <PlayArrowIcon className={classes.playIcon} />
          </IconButton>
          <IconButton aria-label="next">
            {theme.direction === 'rtl' ? (
              <SkipPreviousIcon />
            ) : (
              <SkipNextIcon />
            )}
          </IconButton>
        </div>
      </div>
    </Card>
  )
}

const Api = ({ artId }) => {
  console.log('props', artId)
  try {
    subsonic
      .getArtistInfo(artId)
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        console.log('bedata', { data })
        if (data.status === 'ok') {
          console.log('data', data.artistInfo.biography)
        }
      })
  } catch (error) {
    console.error('err on Artistpage', error)
  }

  return <></>
}

const ArtistView = ({ artist }) => {
  const classes = useStyles()
  console.log('ch is', artist)
  return (
    <Link className={classes.link} to={`/iartist/${artist}`}>
      <div>
        <Api artId={artist} />
        <div>
          <p>This is a Artist Detail page of</p>
          <div>
            <ImgMediaCard />
          </div>
        </div>
      </div>
    </Link>
    // <div>The artist id is </div>
  )
}

export default ArtistView
