import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { useMediaQuery, CircularProgress } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import {
  Button,
  TopToolbar,
  sanitizeListRestProps,
  useDataProvider,
  useNotify,
  useTranslate,
} from 'react-admin'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShareIcon from '@material-ui/icons/Share'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import { IoIosRadio } from 'react-icons/io'
import { playShuffle, playTopSongs } from './actions.js'
import { playSimilar } from '../common/playbackActions.js'
import {
  openShareMenu,
  openDownloadMenu,
  DOWNLOAD_MENU_ARTIST,
} from '../actions'
import config from '../config'
import { formatBytes } from '../utils'
import { artistDownloadSize } from '../common/artist'

const useStyles = makeStyles((theme) => ({
  toolbar: {
    minHeight: 'auto',
    padding: '0 !important',
    background: 'transparent',
    boxShadow: 'none',
    '& .MuiToolbar-root': {
      minHeight: 'auto',
      padding: '0 !important',
      background: 'transparent',
    },
  },
  button: {
    [theme.breakpoints.down('xs')]: {
      minWidth: 'auto',
      padding: '8px 12px',
      fontSize: '0.75rem',
      '& .MuiButton-startIcon': {
        marginRight: '4px',
      },
    },
  },
  radioIcon: {
    [theme.breakpoints.down('xs')]: {
      fontSize: '1.5rem',
    },
  },
}))

const LoadingButton = ({ loading, icon, ...rest }) => (
  <Button {...rest}>
    {loading ? <CircularProgress size={20} color="inherit" /> : icon}
  </Button>
)

const ArtistActions = ({ className, record, ...rest }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const classes = useStyles()
  const isMobile = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const [loadingAction, setLoadingAction] = React.useState(null)
  const isLoading = !!loadingAction

  const albumArtistSize = artistDownloadSize(record)
  const hasAlbumArtistContent = Boolean(albumArtistSize)

  const handlePlay = React.useCallback(async () => {
    setLoadingAction('play')
    try {
      await playTopSongs(dispatch, notify, record.name)
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('Error fetching top songs for artist:', e)
      notify('ra.page.error', 'warning')
    } finally {
      setLoadingAction(null)
    }
  }, [dispatch, notify, record])

  const handleShuffle = React.useCallback(async () => {
    setLoadingAction('shuffle')
    try {
      await playShuffle(dataProvider, dispatch, record.id)
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('Error fetching songs for shuffle:', e)
      notify('ra.page.error', 'warning')
    } finally {
      setLoadingAction(null)
    }
  }, [dataProvider, dispatch, record, notify])

  const handleRadio = React.useCallback(async () => {
    setLoadingAction('radio')
    try {
      await playSimilar(dispatch, notify, record.id)
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('Error starting radio for artist:', e)
      notify('ra.page.error', 'warning')
    } finally {
      setLoadingAction(null)
    }
  }, [dispatch, notify, record])

  const handleShare = React.useCallback(() => {
    dispatch(openShareMenu([record.id], 'artist', record.name))
  }, [dispatch, record])

  const handleDownload = React.useCallback(() => {
    dispatch(openDownloadMenu(record, DOWNLOAD_MENU_ARTIST))
  }, [dispatch, record])

  return (
    <TopToolbar
      className={`${className} ${classes.toolbar}`}
      {...sanitizeListRestProps(rest)}
    >
      <LoadingButton
        onClick={handlePlay}
        label={translate('resources.artist.actions.topSongs')}
        className={classes.button}
        size={isMobile ? 'small' : 'medium'}
        disabled={isLoading}
        loading={loadingAction === 'play'}
        icon={<PlayArrowIcon />}
      />
      <LoadingButton
        onClick={handleShuffle}
        label={translate('resources.artist.actions.shuffle')}
        className={classes.button}
        size={isMobile ? 'small' : 'medium'}
        disabled={isLoading}
        loading={loadingAction === 'shuffle'}
        icon={<ShuffleIcon />}
      />
      <LoadingButton
        onClick={handleRadio}
        label={translate('resources.artist.actions.radio')}
        className={classes.button}
        size={isMobile ? 'small' : 'medium'}
        disabled={isLoading}
        loading={loadingAction === 'radio'}
        icon={<IoIosRadio className={classes.radioIcon} />}
      />
      {config.enableSharing && hasAlbumArtistContent && (
        <LoadingButton
          onClick={handleShare}
          label={translate('ra.action.share')}
          className={classes.button}
          size={isMobile ? 'small' : 'medium'}
          icon={<ShareIcon />}
        />
      )}
      {config.enableDownloads && hasAlbumArtistContent && (
        <LoadingButton
          onClick={handleDownload}
          label={`${translate('ra.action.download')} (${formatBytes(
            albumArtistSize,
          )})`}
          className={classes.button}
          size={isMobile ? 'small' : 'medium'}
          icon={<CloudDownloadOutlinedIcon />}
        />
      )}
    </TopToolbar>
  )
}

ArtistActions.propTypes = {
  className: PropTypes.string,
  record: PropTypes.object.isRequired,
}

ArtistActions.defaultProps = {
  className: '',
}

export default ArtistActions
