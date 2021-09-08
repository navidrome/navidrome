import React, { useState, useEffect, useCallback } from 'react'
import {
  GridList,
  GridListTile,
  Typography,
  Collapse,
  Link,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import withWidth from '@material-ui/core/withWidth'

import subsonic from '../subsonic'

import Card from '@material-ui/core/Card'
import CardContent from '@material-ui/core/CardContent'
import CardMedia from '@material-ui/core/CardMedia'
import PropTypes from 'prop-types'
import Button from '@material-ui/core/Button'

import { AlbumGridTile } from '../album/AlbumGridView'
import { getColsForWidth } from '../album/AlbumGridView'
import {
  useTranslate,
  useShowController,
  ShowContextProvider,
} from 'react-admin'

import { useAlbumsPerPage } from '.'
import { Redirect } from 'react-router'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
      padding: '1em',
      '& .MuiTypography-h5': {
        wordBreak: 'break-word',
      },
      [theme.breakpoints.down('xs')]: {
        padding: 'unset',
        background: ({ img }) => `url(${img})`,
      },
    },
    bgContainer: {
      display: 'flex',
      width: '100%',
      [theme.breakpoints.down('xs')]: {
        height: '15rem',
        width: '100vw',
        padding: 'unset',
        backdropFilter: 'blur(1px)',
        backgroundPosition: '50% 30%',
        background: `linear-gradient(to bottom, rgba(52 52 52 / 72%), rgba(21 21 21))`,
      },
    },
    iroot: {
      margin: '20px',
      display: 'grid',
    },
    details: {
      display: 'flex',
      flex: '1',
      flexDirection: 'column',
    },
    mdetails: {
      display: 'none',
      [theme.breakpoints.down('xs')]: {
        display: 'flex',
        alignItems: 'center',
        width: '7rem',
        marginLeft: '0.5rem',
        flex: '1',
      },
    },
    mbio: {
      display: 'none',
      [theme.breakpoints.down('xs')]: {
        display: 'flex',
        marginLeft: '3%',
        marginRight: '3%',
        zIndex: '1',
      },
    },
    content: {
      flex: '1 0 auto',
      '& .MuiTypography-root': {
        display: ({ expanded }) => (expanded ? 'block' : '-webkit-inline-box'),
        boxOrient: 'vertical',
        lineClamp: '3',
      },
    },
    cover: {
      width: 151,
      boxShadow: '0px 0px 6px 0px #565656',
      borderRadius: '5px',
      [theme.breakpoints.up('sm')]: {
        borderRadius: '7em',
      },
    },
    martImage: {
      marginLeft: '1em',
      maxHeight: '10rem',
      backgroundColor: 'inherit',
      display: 'none',
      [theme.breakpoints.down('xs')]: {
        marginTop: '4rem',
        maxHeight: '7rem',
        width: '7rem',
        display: 'flex',
      },
    },
    artImage: {
      maxHeight: '9.5rem',
      backgroundColor: 'inherit',
      display: 'flex',
      [theme.breakpoints.down('xs')]: {
        marginTop: '4rem',
        maxHeight: '7rem',
        width: '7rem',
      },
    },
    artDetail: {
      flex: '1',
      padding: '3%',
      display: 'flex',
      minHeight: '10rem',
      '& .MuiPaper-elevation1': {
        boxShadow: 'none',
        padding: '4px',
      },
      [theme.breakpoints.down('xs')]: {
        display: 'none',
      },
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
    more: {
      display: 'none',
      [theme.breakpoints.down('xs')]: {
        display: ({ link }) => (link ? 'flex' : 'none'),
        width: '7rem',
        flex: '1',
        alignItems: 'flex-start',
        padding: '0',
        marginBottom: 'auto',
        border: 'none',
        boxShadow: '-10px 0px 18px 5px black',
        background: 'inherit',
        textTransform: 'capitalize',
        '&:hover': {
          background: 'black',
          boxShadow: '-10px 0px 18px 5px black',
        },
        '& .MuiButton-label': {
          color: `${theme.palette.primary.main}`,
        },
      },
    },
    album: {
      marginBottom: '1em',
    },
  }),
  { name: 'NDArtistPage' }
)

function ImgMediaCard({ artId, artist }) {
  const [artisteInfo, setartisteInfo] = useState()
  const [expanded, setExpanded] = useState(false)

  const props = { ...artist }
  const artistProps = props['0']
  let title = artistProps?.artist
  let lastLink = ''
  const link = artisteInfo?.biography?.match(
    /<a\s+(?:[^>]*?\s+)?href=(["'])(.*?)\1/
  )
  const biography = artisteInfo?.biography?.replace(new RegExp('<.*>', 'g'), '')

  const handleExpandClick = useCallback(() => {
    setExpanded(!expanded)
  }, [expanded, setExpanded])

  useEffect(() => {
    subsonic
      .getArtistInfo(artId)
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          setartisteInfo(data.artistInfo)
        }
      })
      .catch((e) => console.error('error on artist page', e))
  }, [artId, artist])

  if (link) {
    lastLink = link[2]
  }

  const img = artisteInfo?.largeImageUrl
  const classes = useStyles({ img, link, expanded })

  return (
    <>
      <div className={classes.root}>
        <div className={classes.bgContainer}>
          <Card className={classes.martImage}>
            <CardMedia
              className={classes.cover}
              image={`${artisteInfo?.mediumImageUrl}`}
              title={title}
            />
          </Card>
          <div className={classes.mdetails}>
            <Typography component="h5" variant="h5">
              {title}
            </Typography>
          </div>
          <Card className={classes.artDetail}>
            <Card className={classes.artImage}>
              <CardMedia
                className={classes.cover}
                image={`${artisteInfo?.mediumImageUrl}`}
                title={title}
              />
            </Card>
            <div className={classes.details}>
              <CardContent className={classes.content}>
                <Typography component="h5" variant="h5">
                  {title}
                </Typography>
                <Collapse
                  collapsedHeight={'4.5em'}
                  in={expanded}
                  timeout={'auto'}
                >
                  <Typography variant={'body1'} onClick={handleExpandClick}>
                    {biography}
                    <Link
                      href={lastLink}
                      style={{ margin: '1px' }}
                      target="_blank"
                      rel="nofollow"
                    >
                      Read more...
                    </Link>
                  </Typography>
                </Collapse>
              </CardContent>
            </div>
          </Card>
        </div>
      </div>
      <div className={classes.mbio}>
        <Collapse collapsedHeight={'1.5em'} in={expanded} timeout={'auto'}>
          <Typography variant={'body1'} onClick={handleExpandClick}>
            {biography}
            <Link href={lastLink} target="_blank" rel="nofollow">
              Read more...
            </Link>
          </Typography>
        </Collapse>
        {!expanded && (
          <Button
            variant="contained"
            color="inherit"
            className={classes.more}
            onClick={handleExpandClick}
          >
            More
          </Button>
        )}
      </div>
    </>
  )
}

const ArtistAlbum = ({ record, width }) => {
  const [artist, setartist] = useState([])
  const classes = useStyles()
  const translate = useTranslate()
  const [perPage] = useAlbumsPerPage(width)
  useEffect(() => {
    subsonic
      .getArtist(record?.id)
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          setartist(data.artist.album.map((s) => [...artist, { ...s }]))
        }
      })
      .catch((e) => {
        console.error('err on ArtistDetail', e)
        return (
          <Redirect
            to={`/album?filter={"artist_id":"${record?.id}"}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}&perPage=${perPage}`}
          />
        )
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [record])

  return (
    <>
      <ImgMediaCard artId={record?.id} artist={artist[0]} />
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

const ArtistShow = (props) => {
  const { width } = props
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <ArtistAlbum width={width} {...props} {...controllerProps} />
    </ShowContextProvider>
  )
}

ArtistShow.propTypes = {
  width: PropTypes.oneOf(['lg', 'md', 'sm', 'xl', 'xs']),
}

export default withWidth()(ArtistShow)
