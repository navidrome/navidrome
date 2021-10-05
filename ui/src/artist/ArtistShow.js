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
import MartistDetails from './mArtsistShow'
import ArtistExternalLinks from './ArtistExternalLink'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
      padding: '1em',
      '& .MuiTypography-h5': {
        wordBreak: 'break-word',
      },
    },
    albumList: {
      margin: '20px',
      display: 'grid',
      [theme.breakpoints.down('xs')]: {
        margin: '0px',
      },
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
    content: {
      flex: '1 0 auto',
    },
    cover: {
      width: 151,
      boxShadow: '0px 0px 6px 0px #565656',
      borderRadius: '5px',
    },
    artImage: {
      maxHeight: '9.5rem',
      backgroundColor: 'inherit',
      display: 'flex',
    },
    artDetail: {
      flex: '1',
      padding: '3%',
      display: 'flex',
      minHeight: '10rem',
      '& .MuiPaper-elevation1': {
        boxShadow: 'none',
        borderRadius: '6em',
      },
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
  }),
  { name: 'NDArtistPage' }
)

const ArtistDetails = ({ record }) => {
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
          <MartistDetails
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
      {record && <ArtistDetails record={record} />}
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
