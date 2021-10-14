import React, { useState, useEffect, useCallback } from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { useMediaQuery, makeStyles } from '@material-ui/core'
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
import ArtistExternalLinks from './ArtistExternalLink'
import config from '../config'
import { LoveButton } from '../common'
import MobileArtistDetails from './MobileArtistDetails'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
      padding: '1em',
    },
    albumList: {
      margin: '20px',
      display: 'grid',
      [theme.breakpoints.down('xs')]: {
        margin: '10px',
      },
    },
    details: {
      display: 'flex',
      flex: '1',
      flexDirection: 'column',
    },
    biography: {
      display: 'inline-block',
      marginTop: '1em',
      float: 'left',
      wordBreak: 'break-word',
      cursor: 'pointer',
    },
    content: {
      flex: '1 0 auto',
    },
    cover: {
      width: 151,
      borderRadius: '6em',
    },
    artistImage: {
      maxHeight: '9.5rem',
      backgroundColor: 'inherit',
      display: 'flex',
      boxShadow: 'none',
    },
    artistDetail: {
      flex: '1',
      padding: '3%',
      display: 'flex',
      minHeight: '10rem',
    },
    artistSummary: {
      marginBottom: '1em',
      [theme.breakpoints.down('xs')]: {
        margin: '2em',
        marginBottom: 'auto',
      },
    },
    button: {
      marginLeft: '0.9em',
    },
    loveButton: {
      top: theme.spacing(-0.2),
      left: theme.spacing(0.5),
    },
    artName: {
      wordBreak: 'break-word',
    },
  }),
  { name: 'NDArtistPage' }
)

const ArtistDetails = (props) => {
  const record = useRecordContext(props)
  const [artistInfo, setArtistInfo] = useState()
  const [expanded, setExpanded] = useState(false)
  const artistId = record?.id

  const title = record.name

  const biography = artistInfo?.biography?.replace(new RegExp('<.*>', 'g'), '')
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('sm'))

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
      {isDesktop ? (
        <div className={classes.root}>
          <Card className={classes.artistDetail}>
            <Card className={classes.artistImage}>
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
                <Typography
                  component="h5"
                  variant="h5"
                  className={classes.artName}
                >
                  {title}
                  {config.enableFavourites && (
                    <LoveButton
                      className={classes.loveButton}
                      record={record}
                      resource={'artist'}
                      size={'default'}
                      aria-label="love"
                      color="primary"
                    />
                  )}
                </Typography>
                <Collapse
                  collapsedHeight={'4.5em'}
                  in={expanded}
                  timeout={'auto'}
                  className={classes.biography}
                >
                  <Typography variant={'body1'} onClick={handleExpandClick}>
                    <span dangerouslySetInnerHTML={{ __html: biography }} />
                  </Typography>
                </Collapse>
              </CardContent>
              <Typography component={'div'} className={classes.button}>
                <ArtistExternalLinks artistInfo={artistInfo} record={record} />
              </Typography>
            </div>
          </Card>
        </div>
      ) : (
        <>
          <MobileArtistDetails
            img={img}
            artistInfo={artistInfo}
            record={record}
            title={title}
            expanded={expanded}
            biography={biography}
            handleExpandClick={handleExpandClick}
          />
        </>
      )}
    </>
  )
}

const ArtistAlbums = ({ ...props }) => {
  const { ids } = props
  const classes = useStyles()
  const translate = useTranslate()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('sm'))

  return (
    <div className={classes.albumList}>
      {isDesktop && (
        <div className={classes.artistSummary}>
          {ids.length +
            ' ' +
            translate('resources.album.name', { smart_count: ids.length })}
        </div>
      )}
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
