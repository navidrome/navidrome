import React, { useMemo, useCallback } from 'react'
import {
  Card,
  CardContent,
  CardMedia,
  Collapse,
  makeStyles,
  Typography,
  useMediaQuery,
} from '@material-ui/core'
import { useTranslate } from 'react-admin'
import clsx from 'clsx'
import Lightbox from 'react-image-lightbox'
import 'react-image-lightbox/style.css'
import subsonic from '../subsonic'
import {
  ArtistLinkField,
  DurationField,
  formatRange,
  SizeField,
  LoveButton,
  RatingField,
} from '../common'
import config from '../config'
import {Link} from 'react-admin'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      [theme.breakpoints.down('xs')]: {
        padding: '0.7em',
        minWidth: '20em',
      },
      [theme.breakpoints.up('sm')]: {
        padding: '1em',
        minWidth: '32em',
      },
    },
    cardContents: {
      display: 'flex',
    },
    details: {
      display: 'flex',
      flexDirection: 'column',
    },
    content: {
      flex: '2 0 auto',
    },
    coverParent: {
      [theme.breakpoints.down('xs')]: {
        height: '8em',
        width: '8em',
        minWidth: '8em',
      },
      [theme.breakpoints.up('sm')]: {
        height: '10em',
        width: '10em',
        minWidth: '10em',
      },
      [theme.breakpoints.up('lg')]: {
        height: '15em',
        width: '15em',
        minWidth: '15em',
      },
    },
    cover: {
      objectFit: 'contain',
      cursor: 'pointer',
      display: 'block',
      width: '100%',
      height: '100%',
    },
    loveButton: {
      top: theme.spacing(-0.2),
      left: theme.spacing(0.5),
    },
    commentBlock: {
      display: 'inline-block',
      marginTop: '1em',
      float: 'left',
      wordBreak: 'break-all',
    },
    pointerCursor: {
      cursor: 'pointer',
    },
    recordName: {},
    recordArtist: {},
    recordMeta: {},
  }),
  {
    name: 'NDAlbumDetails',
  }
)

const AlbumComment = ({ record }) => {
  const classes = useStyles()
  const [expanded, setExpanded] = React.useState(false)

  const lines = record.comment.split('\n')
  const formatted = useMemo(() => {
    return lines.map((line, idx) => (
      <span key={record.id + '-comment-' + idx}>
        <span dangerouslySetInnerHTML={{ __html: line }} />
        <br />
      </span>
    ))
  }, [lines, record.id])

  
    const handleExpandClick = useCallback(() => {
    setExpanded(!expanded)
  }, [expanded, setExpanded])

  return (
    <Collapse
      collapsedHeight={'1.5em'}
      in={expanded}
      timeout={'auto'}
      className={clsx(
        classes.commentBlock,
        lines.length > 1 && classes.pointerCursor
      )}
    >
      <Typography variant={'body1'} onClick={handleExpandClick}>
        {formatted}
      </Typography>
    </Collapse>
  )
}


const CommentLink = ({record}) => {
    const urlRegx = /(https?:\/\/)?[\w\-~]+(\.[\w\-~]+)+(\/[\w\-~@:%]*)*(#[\w\-]*)?(\?[^\s]*)?/gi
    const result =record.comment.match(urlRegx)
    const handleOpenLink= () =>{
    window.open(result[0], '_blank');
    }
    
      return (
      <Link 
      onClick={handleOpenLink}
      >
      {record.comment}
      </Link>

  )
}


const AlbumDetails = ({ record }) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('lg'))
  const classes = useStyles()
  const [isLightboxOpen, setLightboxOpen] = React.useState(false)
  const translate = useTranslate()

  const genreYear = (record) => {
    let genreDateLine = []
    if (record.genre) {
      genreDateLine.push(record.genre)
    }
    const year = formatRange(record, 'year')
    if (year) {
      genreDateLine.push(year)
    }
    return genreDateLine.join(' · ')
  }

  const imageUrl = subsonic.getCoverArtUrl(record, 300)
  const fullImageUrl = subsonic.getCoverArtUrl(record)

  const handleOpenLightbox = React.useCallback(() => setLightboxOpen(true), [])
  const handleCloseLightbox = React.useCallback(
    () => setLightboxOpen(false),
    []
  )
  return (
    <Card className={classes.root}>
      <div className={classes.cardContents}>
        <div className={classes.coverParent}>
          <CardMedia
            component={'img'}
            src={imageUrl}
            width="400"
            height="400"
            className={classes.cover}
            onClick={handleOpenLightbox}
            title={record.name}
          />
        </div>
        <div className={classes.details}>
          <CardContent className={classes.content}>
            <Typography
              variant={isDesktop ? 'h5' : 'h6'}
              className={classes.recordName}
            >
              {record.name}
              {config.enableFavourites && (
                <LoveButton
                  className={classes.loveButton}
                  record={record}
                  resource={'album'}
                  size={isDesktop ? 'default' : 'small'}
                  aria-label="love"
                  color="primary"
                />
              )}
            </Typography>
            <Typography component="h6" className={classes.recordArtist}>
              <ArtistLinkField record={record} />
            </Typography>
            <Typography component="p" className={classes.recordMeta}>
              {genreYear(record)}
            </Typography>
            <Typography component="p" className={classes.recordMeta}>
              {record.songCount}{' '}
              {translate('resources.song.name', {
                smart_count: record.songCount,
              })}
              {' · '} <DurationField record={record} source={'duration'} />{' '}
              {' · '}
              <SizeField record={record} source="size" />
            </Typography>
            {config.enableStarRating && (
              <div>
                <RatingField
                  record={record}
                  resource={'album'}
                  size={isDesktop ? 'medium' : 'small'}
                />
              </div>
            )}
            {isDesktop && record['comment'] && <CommentLink record={record}/>}
          </CardContent>
        </div>
      </div>
      {!isDesktop && record['comment'] && <CommentLink record={record}/> }
      {isLightboxOpen && (
        <Lightbox
          imagePadding={50}
          animationDuration={200}
          imageTitle={record.name}
          mainSrc={fullImageUrl}
          onCloseRequest={handleCloseLightbox}
        />
      )}
    </Card>
  )
}

export default AlbumDetails
