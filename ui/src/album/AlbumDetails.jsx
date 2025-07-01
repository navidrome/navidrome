import { useCallback, useEffect, useState } from 'react'
import {
  Card,
  CardContent,
  CardMedia,
  Collapse,
  makeStyles,
  Typography,
  useMediaQuery,
  withWidth,
} from '@material-ui/core'
import {
  ArrayField,
  ChipField,
  Link,
  SingleFieldList,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import Lightbox from 'react-image-lightbox'
import 'react-image-lightbox/style.css'
import subsonic from '../subsonic'
import {
  ArtistLinkField,
  CollapsibleComment,
  DurationField,
  formatRange,
  LoveButton,
  RatingField,
  SizeField,
  useAlbumsPerPage,
} from '../common'
import config from '../config'
import { formatFullDate, intersperse } from '../utils'
import AlbumExternalLinks from './AlbumExternalLinks'

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
      backgroundColor: 'transparent',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    },
    cover: {
      objectFit: 'contain',
      cursor: 'pointer',
      display: 'block',
      width: '100%',
      height: '100%',
      backgroundColor: 'transparent',
      transition: 'opacity 0.3s ease-in-out',
    },
    coverLoading: {
      opacity: 0.5,
    },
    loveButton: {
      top: theme.spacing(-0.2),
      left: theme.spacing(0.5),
    },
    notes: {
      display: 'inline-block',
      marginTop: '1em',
      float: 'left',
      wordBreak: 'break-word',
      cursor: 'pointer',
    },
    recordName: {},
    recordArtist: {},
    recordMeta: {},
    genreList: {
      marginTop: theme.spacing(0.5),
    },
    externalLinks: {
      marginTop: theme.spacing(1.5),
    },
  }),
  {
    name: 'NDAlbumDetails',
  },
)

const useGetHandleGenreClick = (width) => {
  const [perPage] = useAlbumsPerPage(width)

  return (id) => {
    return `/album?filter={"genre_id":["${id}"]}&order=ASC&sort=name&perPage=${perPage}`
  }
}

const GenreChipField = withWidth()(({ width, ...rest }) => {
  const record = useRecordContext(rest)
  const genreLink = useGetHandleGenreClick(width)

  return (
    <Link to={genreLink(record.id)} onClick={(e) => e.stopPropagation()}>
      <ChipField
        source="name"
        // Workaround to force ChipField to be clickable
        onClick={() => {}}
      />
    </Link>
  )
})

const GenreList = () => {
  const classes = useStyles()
  return (
    <ArrayField className={classes.genreList} source={'genres'}>
      <SingleFieldList linkType={false}>
        <GenreChipField />
      </SingleFieldList>
    </ArrayField>
  )
}

export const Details = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const translate = useTranslate()
  const record = useRecordContext(props)

  // Create an array of detail elements
  let details = []
  const addDetail = (obj) => {
    const id = details.length
    details.push(<span key={`detail-${record.id}-${id}`}>{obj}</span>)
  }

  // Calculate date related fields
  const yearRange = formatRange(record, 'year')
  const date = record.date ? formatFullDate(record.date) : yearRange

  const originalDate = record.originalDate
    ? formatFullDate(record.originalDate)
    : formatRange(record, 'originalYear')
  const releaseDate = record?.releaseDate && formatFullDate(record.releaseDate)

  const dateToUse = originalDate || date
  const isOriginalDate = originalDate && dateToUse !== date
  const showDate = dateToUse && dateToUse !== releaseDate

  // Get label for the main date display
  const getDateLabel = () => {
    if (isXsmall) return '♫'
    if (isOriginalDate) return translate('resources.album.fields.originalDate')
    return null
  }

  // Get label for release date display
  const getReleaseDateLabel = () => {
    if (!isXsmall) return translate('resources.album.fields.releaseDate')
    if (showDate) return '○'
    return null
  }

  // Display dates with appropriate labels
  if (showDate) {
    addDetail(<>{[getDateLabel(), dateToUse].filter(Boolean).join('  ')}</>)
  }

  if (releaseDate) {
    addDetail(
      <>{[getReleaseDateLabel(), releaseDate].filter(Boolean).join('  ')}</>,
    )
  }
  addDetail(
    <>
      {record.songCount +
        ' ' +
        translate('resources.song.name', {
          smart_count: record.songCount,
        })}
    </>,
  )
  !isXsmall && addDetail(<DurationField source={'duration'} />)
  !isXsmall && addDetail(<SizeField source="size" />)

  // Return the details rendered with separators
  return <>{intersperse(details, ' · ')}</>
}

const AlbumDetails = (props) => {
  const record = useRecordContext(props)
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('lg'))
  const classes = useStyles()
  const [isLightboxOpen, setLightboxOpen] = useState(false)
  const [expanded, setExpanded] = useState(false)
  const [albumInfo, setAlbumInfo] = useState()
  const [imageLoading, setImageLoading] = useState(false)
  const [imageError, setImageError] = useState(false)

  let notes =
    albumInfo?.notes?.replace(new RegExp('<.*>', 'g'), '') || record.notes

  if (notes !== undefined) {
    notes += '..'
  }

  useEffect(() => {
    subsonic
      .getAlbumInfo(record.id)
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          setAlbumInfo(data.albumInfo)
        }
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.error('error on album page', e)
      })
  }, [record])

  // Reset image state when album changes
  useEffect(() => {
    setImageLoading(true)
    setImageError(false)
  }, [record.id])

  const imageUrl = subsonic.getCoverArtUrl(record, 300)
  const fullImageUrl = subsonic.getCoverArtUrl(record)

  const handleImageLoad = useCallback(() => {
    setImageLoading(false)
    setImageError(false)
  }, [])

  const handleImageError = useCallback(() => {
    setImageLoading(false)
    setImageError(true)
  }, [])

  const handleOpenLightbox = useCallback(() => {
    if (!imageError) {
      setLightboxOpen(true)
    }
  }, [imageError])

  const handleCloseLightbox = useCallback(() => setLightboxOpen(false), [])

  return (
    <Card className={classes.root}>
      <div className={classes.cardContents}>
        <div className={classes.coverParent}>
          <CardMedia
            key={record.id}
            component={'img'}
            src={imageUrl}
            width="400"
            height="400"
            className={`${classes.cover} ${imageLoading ? classes.coverLoading : ''}`}
            onClick={handleOpenLightbox}
            onLoad={handleImageLoad}
            onError={handleImageError}
            title={record.name}
            style={{
              cursor: imageError ? 'default' : 'pointer',
            }}
          />
        </div>
        <div className={classes.details}>
          <CardContent className={classes.content}>
            <Typography
              variant={isDesktop ? 'h5' : 'h6'}
              className={classes.recordName}
            >
              {record.name}
              <LoveButton
                className={classes.loveButton}
                record={record}
                resource={'album'}
                size={isDesktop ? 'default' : 'small'}
                aria-label="love"
                color="primary"
              />
            </Typography>
            <Typography component={'h6'} className={classes.recordArtist}>
              {record?.tags?.['albumversion']}
            </Typography>
            <Typography component={'h6'} className={classes.recordArtist}>
              <ArtistLinkField record={record} />
            </Typography>
            <Typography component={'div'} className={classes.recordMeta}>
              <Details />
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
            {isDesktop ? (
              <GenreList />
            ) : (
              <Typography component={'p'}>{record.genre}</Typography>
            )}
            {!isXsmall && (
              <Typography component={'div'} className={classes.recordMeta}>
                {config.enableExternalServices && (
                  <AlbumExternalLinks className={classes.externalLinks} />
                )}
              </Typography>
            )}
            {isDesktop && (
              <Collapse
                collapsedHeight={'2.75em'}
                in={expanded}
                timeout={'auto'}
                className={classes.notes}
              >
                <Typography
                  variant={'body1'}
                  onClick={() => setExpanded(!expanded)}
                >
                  <span dangerouslySetInnerHTML={{ __html: notes }} />
                </Typography>
              </Collapse>
            )}
            {isDesktop && record['comment'] && (
              <CollapsibleComment record={record} />
            )}
          </CardContent>
        </div>
      </div>
      {!isDesktop && record['comment'] && (
        <CollapsibleComment record={record} />
      )}
      {!isDesktop && (
        <div className={classes.notes}>
          <Collapse collapsedHeight={'1.5em'} in={expanded} timeout={'auto'}>
            <Typography
              variant={'body1'}
              onClick={() => setExpanded(!expanded)}
            >
              <span dangerouslySetInnerHTML={{ __html: notes }} />
            </Typography>
          </Collapse>
        </div>
      )}
      {isLightboxOpen && !imageError && (
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
