import React, { useState, useEffect, useCallback } from 'react'
import {
  GridList,
  GridListTile,
  Typography,
  Collapse,
  withWidth,
  Link,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'

import subsonic from '../subsonic'

import Card from '@material-ui/core/Card'
import CardContent from '@material-ui/core/CardContent'
import CardMedia from '@material-ui/core/CardMedia'
import PropTypes from 'prop-types'
import ExpandMoreIcon from '@material-ui/icons/ExpandMore'
import ExpandLessIcon from '@material-ui/icons/ExpandLess'
import Button from '@material-ui/core/Button'

import { AlbumGridTile } from '../album/AlbumGridView'
import { getColsForWidth } from '../album/AlbumGridView'
import { useTranslate } from 'react-admin'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
      padding: '1em',
    },
    iroot: {
      margin: '20px',
      display: 'grid',
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
      boxShadow: '1px 1px 20px 0px #565656',
      borderRadius: '5px',
    },
    artImage: {
      marginTop: '1rem',
      maxHeight: '10rem',
      backgroundColor: 'inherit',
      display: 'flex',
    },
    artDetail: {
      margin: '1rem',
      flex: '1',
      display: 'flex',
      minHeight: '10rem',
    },
    expand: {
      display: 'flex',
      padding: '0',
      boxShadow: 'none',
      backgroundColor: 'inherit',
      fontSize: '0.77rem',
      color: '#a0a0a0',
      border: 'none',
      '& .MuiButton-label': {
        display: 'contents',
      },
      '&.MuiButton-contained:hover': {
        boxShadow: 'none',
        backgroundColor: 'inherit !important',
        color: '#dbdada',
      },
    },
    album: {
      marginBottom: '1em',
    },
  }),
  { name: 'NDArtistPage' }
)

function ImgMediaCard({ artId, artist }) {
  const classes = useStyles()
  const [lastInfo, setlastInfo] = useState()
  const [expanded, setExpanded] = useState(false)

  const props = { ...artist }
  const artistProps = props['0']
  var title = artistProps?.artist
  var lastLink = ''

  const link = lastInfo?.biography?.match(
    /<a\s+(?:[^>]*?\s+)?href=(["'])(.*?)\1/
  )
  const biography = lastInfo?.biography?.replace(new RegExp('<.*>', 'g'), '')

  try {
    useEffect(() => {
      subsonic
        .getArtistInfo(artId)
        .then((resp) => resp.json['subsonic-response'])
        .then((data) => {
          if (data.status === 'ok') {
            setlastInfo(data.artistInfo)
          }
        })
    }, [artId])
  } catch (error) {
    console.error('err on Artistpage', error)
  }

  if (link != undefined) {
    lastLink = link[2]
  }

  const handleExpandClick = useCallback(() => {
    setExpanded(!expanded)
  }, [expanded, setExpanded])

  return (
    <div classsName={classes.root} style={{ display: 'flex' }}>
      <Card className={classes.artImage}>
        <CardMedia
          className={classes.cover}
          image={`${lastInfo?.mediumImageUrl}`}
          title={title}
        />
      </Card>
      <Card className={classes.artDetail}>
        <div className={classes.details}>
          <CardContent className={classes.content}>
            <Typography component="h5" variant="h5">
              {title}
            </Typography>
            <Collapse collapsedHeight={'1.5em'} in={expanded} timeout={'auto'}>
              <Typography variant={'body1'} onClick={handleExpandClick}>
                {biography}
                <Link href={lastLink} target="_blank" rel="nofollow">
                  Read more...
                </Link>
              </Typography>
            </Collapse>
            {expanded ? (
              <Button
                variant="contained"
                color="inherit"
                className={classes.expand}
                endIcon={<ExpandLessIcon />}
                onClick={handleExpandClick}
              >
                Read less
              </Button>
            ) : (
              <Button
                variant="contained"
                color="inherit"
                className={classes.expand}
                endIcon={<ExpandMoreIcon />}
                onClick={handleExpandClick}
              >
                Read More
              </Button>
            )}
          </CardContent>
        </div>
      </Card>
    </div>
  )
}

const ArtistAlbum = ({ artId, width }) => {
  const [artist, setartist] = useState([])
  const classes = useStyles()
  const translate = useTranslate()
  try {
    useEffect(() => {
      subsonic
        .getArtist(artId)
        .then((resp) => resp.json['subsonic-response'])
        .then((data) => {
          if (data.status === 'ok') {
            setartist(data.artist.album.map((s) => [...artist, { ...s }]))
          }
        })
    }, [artId])
  } catch (error) {
    console.error('err on Artistpage', error)
  }

  return (
    <>
      <ImgMediaCard artId={artId} artist={artist[0]} />
      <div className={classes.iroot}>
        <div className={classes.album}>
          {artist.length +
            ' ' +
            translate('resources.album.name', { smart_count: artist.length })}
        </div>
        <GridList
          component={'div'}
          cellHeight={'auto'}
          cols={getColsForWidth(width)}
          spacing={20}
        >
          {artist.map((artist) => (
            <GridListTile className={classes.gridListTile} key={artist[0].id}>
              <AlbumGridTile
                record={artist[0]}
                basePath={'/album'}
                showArtist={true}
              />
            </GridListTile>
          ))}
        </GridList>
      </div>
    </>
  )
}

const ArtistView = ({ artist, width }) => {
  return <ArtistAlbum artId={artist} width={width} />
}

ArtistView.propTypes = {
  width: PropTypes.oneOf(['lg', 'md', 'sm', 'xl', 'xs']),
}

export default withWidth()(ArtistView)
