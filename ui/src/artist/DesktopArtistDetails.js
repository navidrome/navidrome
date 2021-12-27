import React, { useState } from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core'
import Card from '@material-ui/core/Card'
import CardContent from '@material-ui/core/CardContent'
import CardMedia from '@material-ui/core/CardMedia'
import ArtistExternalLinks from './ArtistExternalLink'
import config from '../config'
import { LoveButton, RatingField } from '../common'
import Lightbox from 'react-image-lightbox'

import Accordion from '@material-ui/core/Accordion'
import AccordionSummary from '@material-ui/core/AccordionSummary'
import AccordionDetails from '@material-ui/core/AccordionDetails'
import { ExpandMore } from '@material-ui/icons'
import { ReferenceManyField } from 'react-admin'
import AlbumSongs from '../album/AlbumSongs'
import AlbumActions from '../album/AlbumActions'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
      flexDirection: 'column',
      padding: '1em',
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
      cursor: 'pointer',
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
    button: {
      marginLeft: '0.9em',
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
    albumActions: {
      display: 'none',
    },
    accordion: {
      flexDirection: 'column',
      '& > :first-child': {
        display: 'none!important',
      },
    },
    expanded: {
      background: 'inherit',
    },
  }),
  { name: 'NDDesktopArtistDetails' }
)

const DesktopArtistDetails = ({
  img,
  artistInfo,
  record,
  biography,
  topSong,
  showContext,
}) => {
  const [expanded, setExpanded] = useState(false)
  const classes = useStyles({ img, expanded })
  const title = record.name
  const [isLightboxOpen, setLightboxOpen] = React.useState(false)

  const handleOpenLightbox = React.useCallback(() => setLightboxOpen(true), [])
  const handleCloseLightbox = React.useCallback(
    () => setLightboxOpen(false),
    []
  )

  let ids = []

  topSong && topSong.map((sng) => ids.push(sng.id))

  return (
    <div className={classes.root}>
      <Card className={classes.artistDetail}>
        <Card className={classes.artistImage}>
          {artistInfo && (
            <CardMedia
              className={classes.cover}
              image={artistInfo.mediumImageUrl}
              onClick={handleOpenLightbox}
              title={title}
            />
          )}
        </Card>
        <div className={classes.details}>
          <CardContent className={classes.content}>
            <Typography
              component="h5"
              variant="h5"
              className={classes.artistName}
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
            {config.enableStarRating && (
              <div>
                <RatingField
                  record={record}
                  resource={'artist'}
                  size={'small'}
                  className={classes.rating}
                />
              </div>
            )}
            <Collapse
              collapsedHeight={'4.5em'}
              in={expanded}
              timeout={'auto'}
              className={classes.biography}
            >
              <Typography
                variant={'body1'}
                onClick={() => setExpanded(!expanded)}
              >
                <span dangerouslySetInnerHTML={{ __html: biography }} />
              </Typography>
            </Collapse>
          </CardContent>
          <Typography component={'div'} className={classes.button}>
            <ArtistExternalLinks artistInfo={artistInfo} record={record} />
          </Typography>
        </div>
        {isLightboxOpen && (
          <Lightbox
            imagePadding={50}
            animationDuration={200}
            imageTitle={record.name}
            mainSrc={artistInfo.largeImageUrl}
            onCloseRequest={handleCloseLightbox}
          />
        )}
      </Card>

      {topSong && (
        <Accordion classes={{ expanded: classes.expanded }}>
          <AccordionSummary
            expandIcon={<ExpandMore />}
            aria-controls="panel1a-content"
            id="panel1a-header"
          >
            <Typography>Top Songs</Typography>
          </AccordionSummary>
          <AccordionDetails className={classes.accordion}>
            <TopSongs
              showContext={showContext}
              topSong={topSong}
              record={record}
            />
          </AccordionDetails>
        </Accordion>
      )}
    </div>
  )
}

export const TopSongs = ({ showContext, topSong, record }) => {
  const classes = useStyles()
  let ids = []

  topSong && topSong.map((sng) => ids.push(sng.id))
  return (
    <>
      {record && (
        <ReferenceManyField
          {...showContext}
          addLabel={false}
          reference="song"
          target="artist_id"
          sort={{ field: 'title', order: 'ASC' }}
          perPage={0}
          filter={{ id: ids }}
          pagination={null}
        >
          <AlbumSongs
            resource={'album'}
            exporter={false}
            album={record}
            show={false}
            actions={
              <AlbumActions className={classes.albumActions} record={record} />
            }
          />
        </ReferenceManyField>
      )}
    </>
  )
}

export default DesktopArtistDetails
