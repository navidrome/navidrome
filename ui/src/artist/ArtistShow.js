import React, { useState, useEffect, useCallback } from 'react'
import { Typography, Collapse, Link } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import Card from '@material-ui/core/Card'
import CardContent from '@material-ui/core/CardContent'
import CardMedia from '@material-ui/core/CardMedia'
import {
  useTranslate,
  useShowController,
  ShowContextProvider,
  useRecordContext,
  useShowContext,
  ReferenceManyField,
} from 'react-admin'
import subsonic from '../subsonic'
import AlbumGridView from '../album/AlbumGridView'

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
    albumList: {
      margin: '20px',
      display: 'grid',
    },
    details: {
      display: 'flex',
      flex: '1',
      flexDirection: 'column',
    },
    bioBlock: {
      display: 'inline-block',
      marginTop: '1em',
      float: 'left',
      wordBreak: 'break-word',
      cursor: 'pointer',
    },
    link: {
      margin: '1px',
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
        '& p': {
          whiteSpace: ({ expanded }) => (expanded ? 'unset' : 'nowrap'),
          overflow: 'hidden',
          width: '95vw',
          textOverflow: 'ellipsis',
        },
      },
    },
    content: {
      flex: '1 0 auto',
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
    artistSummary: {
      marginBottom: '1em',
    },
  }),
  { name: 'NDArtistPage' }
)

const ArtistDetails = () => {
  const [artistInfo, setArtistInfo] = useState()
  const [expanded, setExpanded] = useState(false)
  const record = useRecordContext()
  const artistId = record?.id

  const title = record.name
  let completeBioLink = ''
  const link = artistInfo?.biography?.match(
    /<a\s+(?:[^>]*?\s+)?href=(["'])(.*?)\1/
  )
  if (link) {
    completeBioLink = link[2]
  }
  const biography = artistInfo?.biography?.replace(new RegExp('<.*>', 'g'), '')
  const translate = useTranslate()

  const img = artistInfo?.largeImageUrl
  const classes = useStyles({ img, expanded })

  useEffect(() => {
    subsonic
      .getArtistInfo(artistId)
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          setArtistInfo(data.artistInfo)
        }
      })
      .catch((e) => {
        console.error('error on artist page', e)
      })
  }, [artistId, record])

  const handleExpandClick = useCallback(() => {
    setExpanded(!expanded)
  }, [expanded, setExpanded])

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
          <Card className={classes.artDetail}>
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
              <CardContent className={classes.content}>
                <Typography component="h5" variant="h5">
                  {title}
                </Typography>
                <Collapse
                  collapsedHeight={'4.5em'}
                  in={expanded}
                  timeout={'auto'}
                  className={classes.bioBlock}
                >
                  <Typography variant={'body1'} onClick={handleExpandClick}>
                    <span dangerouslySetInnerHTML={{ __html: biography }} />
                    {completeBioLink !== '' && (
                      <Link
                        href={completeBioLink}
                        className={classes.link}
                        target="_blank"
                        rel="nofollow"
                      >
                        {translate('message.lastfmLink')}
                      </Link>
                    )}
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

const ArtistAlbums = ({ ...props }) => {
  const { ids } = props
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <div className={classes.albumList}>
      <div className={classes.artistSummary}>
        {ids.length +
          ' ' +
          translate('resources.album.name', { smart_count: ids.length })}
      </div>
      <AlbumGridView {...props} />
    </div>
  )
}

const AlbumShowLayout = (props) => {
  const showContext = useShowContext(props)
  const record = useRecordContext()

  return (
    <>
      {record && <ArtistDetails />}
      {record && (
        <ReferenceManyField
          {...showContext}
          addLabel={false}
          reference="album"
          target="artist_id"
          sort={{ field: 'maxYear', order: 'ASC' }}
          filter={{ artist_id: record?.id }}
          perPage={0}
          pagination={null}
        >
          <ArtistAlbums />
        </ReferenceManyField>
      )}
    </>
  )
}

const ArtistShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <AlbumShowLayout {...controllerProps} />
    </ShowContextProvider>
  )
}

export default ArtistShow
