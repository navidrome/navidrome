import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { useMediaQuery } from '@material-ui/core'
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
import { IoIosRadio } from 'react-icons/io'
import { playTracks } from '../actions'
import { playSimilar } from '../utils'
import subsonic from '../subsonic'

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

const ArtistActions = ({ className, record, ...rest }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const classes = useStyles()
  const isMobile = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  const handlePlay = React.useCallback(async () => {
    try {
      const res = await subsonic.getTopSongs(record.name, 50)
      const data = res.json['subsonic-response']

      if (data.status !== 'ok') {
        throw new Error(
          `Error fetching top songs: ${data.error?.message || 'Unknown error'} (Code: ${data.error?.code || 'unknown'})`,
        )
      }

      const songs = data.topSongs?.song || []
      if (!songs.length) {
        notify('message.noTopSongsFound', 'warning')
        return
      }

      const songData = {}
      const ids = []
      songs.forEach((s) => {
        songData[s.id] = s
        ids.push(s.id)
      })
      dispatch(playTracks(songData, ids))
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('Error fetching top songs for artist:', e)
      notify('ra.page.error', 'warning')
    }
  }, [dispatch, notify, record])

  const handleShuffle = React.useCallback(async () => {
    try {
      const res = await dataProvider.getList('song', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'random', order: 'ASC' },
        filter: { album_artist_id: record.id, missing: false },
      })

      const data = {}
      const ids = []
      res.data.forEach((s) => {
        data[s.id] = s
        ids.push(s.id)
      })
      dispatch(playTracks(data, ids))
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('Error fetching songs for shuffle:', e)
      notify('ra.page.error', 'warning')
    }
  }, [dataProvider, dispatch, record, notify])

  const handleRadio = React.useCallback(async () => {
    try {
      await playSimilar(dispatch, notify, record.id)
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('Error starting radio for artist:', e)
      notify('ra.page.error', 'warning')
    }
  }, [dispatch, notify, record])

  return (
    <TopToolbar
      className={`${className} ${classes.toolbar}`}
      {...sanitizeListRestProps(rest)}
    >
      <Button
        onClick={handlePlay}
        label={translate('resources.artist.actions.play')}
        className={classes.button}
        size={isMobile ? 'small' : 'medium'}
      >
        <PlayArrowIcon />
      </Button>
      <Button
        onClick={handleShuffle}
        label={translate('resources.artist.actions.shuffle')}
        className={classes.button}
        size={isMobile ? 'small' : 'medium'}
      >
        <ShuffleIcon />
      </Button>
      <Button
        onClick={handleRadio}
        label={translate('resources.artist.actions.radio')}
        className={classes.button}
        size={isMobile ? 'small' : 'medium'}
      >
        <IoIosRadio className={classes.radioIcon} />
      </Button>
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
