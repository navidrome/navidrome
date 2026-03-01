import {
  Card,
  CardContent,
  CardMedia,
  IconButton,
  Tooltip,
  Typography,
  useMediaQuery,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import PhotoCameraIcon from '@material-ui/icons/PhotoCamera'
import DeleteIcon from '@material-ui/icons/Delete'
import { useTranslate, useNotify, useRefresh } from 'react-admin'
import { useCallback, useRef, useState, useEffect } from 'react'
import Lightbox from 'react-image-lightbox'
import 'react-image-lightbox/style.css'
import {
  CollapsibleComment,
  DurationField,
  SizeField,
  isWritable,
} from '../common'
import subsonic from '../subsonic'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'

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
      position: 'relative',
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
    coverOverlay: {
      position: 'absolute',
      bottom: 0,
      right: 0,
      display: 'flex',
      gap: '2px',
      padding: '2px',
      backgroundColor: 'rgba(0,0,0,0.5)',
      borderRadius: '4px 0 0 0',
      opacity: 0,
      transition: 'opacity 0.2s ease-in-out',
      '$coverParent:hover &': {
        opacity: 1,
      },
    },
    overlayButton: {
      color: '#fff',
      padding: '4px',
      '&:hover': {
        backgroundColor: 'rgba(255,255,255,0.2)',
      },
    },
    overlayIcon: {
      fontSize: '1.2rem',
    },
    title: {
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      wordBreak: 'break-word',
    },
    stats: {
      marginTop: '1em',
      marginBottom: '0.5em',
    },
  }),
  {
    name: 'NDPlaylistDetails',
  },
)

const PlaylistDetails = (props) => {
  const { record = {} } = props
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('lg'))
  const [isLightboxOpen, setLightboxOpen] = useState(false)
  const [imageLoading, setImageLoading] = useState(false)
  const [imageError, setImageError] = useState(false)
  const fileInputRef = useRef(null)

  const imageUrl = subsonic.getCoverArtUrl(record, 300, true)
  const fullImageUrl = subsonic.getCoverArtUrl(record)
  const canEdit = isWritable(record.ownerId)

  // Reset image state when playlist changes
  useEffect(() => {
    setImageLoading(true)
    setImageError(false)
  }, [record.id])

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

  const handleUploadClick = useCallback(
    (e) => {
      e.stopPropagation()
      if (fileInputRef.current) {
        fileInputRef.current.click()
      }
    },
    [fileInputRef],
  )

  const handleFileChange = useCallback(
    async (e) => {
      const file = e.target.files[0]
      if (!file || !record.id) return

      const formData = new FormData()
      formData.append('image', file)

      try {
        await httpClient(`${REST_URL}/playlist/${record.id}/image`, {
          method: 'POST',
          headers: new Headers({}),
          body: formData,
        })
        notify('resources.playlist.message.coverUploaded', 'success')
        refresh()
      } catch (err) {
        notify('resources.playlist.message.coverUploadError', 'warning')
      }

      // Reset file input so the same file can be re-selected
      e.target.value = ''
    },
    [record.id, notify, refresh],
  )

  const handleRemoveCover = useCallback(
    async (e) => {
      e.stopPropagation()
      if (!record.id) return

      try {
        await httpClient(`${REST_URL}/playlist/${record.id}/image`, {
          method: 'DELETE',
        })
        notify('resources.playlist.message.coverRemoved', 'success')
        refresh()
      } catch (err) {
        notify('resources.playlist.message.coverRemoveError', 'warning')
      }
    },
    [record.id, notify, refresh],
  )

  return (
    <Card className={classes.root}>
      <div className={classes.cardContents}>
        <div className={classes.coverParent}>
          <CardMedia
            key={record.id} // Force re-render when playlist changes
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
          {canEdit && (
            <div className={classes.coverOverlay}>
              <Tooltip
                title={translate('resources.playlist.actions.uploadCover')}
              >
                <IconButton
                  className={classes.overlayButton}
                  onClick={handleUploadClick}
                  size="small"
                >
                  <PhotoCameraIcon className={classes.overlayIcon} />
                </IconButton>
              </Tooltip>
              {record.imageFile && (
                <Tooltip
                  title={translate('resources.playlist.actions.removeCover')}
                >
                  <IconButton
                    className={classes.overlayButton}
                    onClick={handleRemoveCover}
                    size="small"
                  >
                    <DeleteIcon className={classes.overlayIcon} />
                  </IconButton>
                </Tooltip>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                style={{ display: 'none' }}
                onChange={handleFileChange}
              />
            </div>
          )}
        </div>
        <div className={classes.details}>
          <CardContent className={classes.content}>
            <Typography
              variant={isDesktop ? 'h5' : 'h6'}
              className={classes.title}
            >
              {record.name || translate('ra.page.loading')}
            </Typography>
            <Typography component="p" className={classes.stats}>
              {record.songCount ? (
                <span>
                  {record.songCount}{' '}
                  {translate('resources.song.name', {
                    smart_count: record.songCount,
                  })}
                  {' · '}
                  <DurationField record={record} source={'duration'} />
                  {' · '}
                  <SizeField record={record} source={'size'} />
                </span>
              ) : (
                <span>&nbsp;</span>
              )}
            </Typography>
            <CollapsibleComment record={record} />
          </CardContent>
        </div>
      </div>
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

export default PlaylistDetails
