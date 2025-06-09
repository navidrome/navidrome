import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import {
  Button,
  TopToolbar,
  sanitizeListRestProps,
  useDataProvider,
  useNotify,
  useTranslate,
} from 'react-admin'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import { IoIosRadio } from 'react-icons/io'
import { playTracks } from '../actions'
import { playSimilar } from '../utils'

const ArtistActions = ({ className, record, ...rest }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()

  const handleShuffle = React.useCallback(() => {
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'random', order: 'ASC' },
        filter: { album_artist_id: record.id, missing: false },
      })
      .then((res) => {
        const data = {}
        const ids = []
        res.data.forEach((s) => {
          data[s.id] = s
          ids.push(s.id)
        })
        dispatch(playTracks(data, ids))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }, [dataProvider, dispatch, record, notify])

  const handleRadio = React.useCallback(async () => {
    try {
      await playSimilar(dispatch, notify, record.id)
    } catch {
      notify('ra.page.error', 'warning')
    }
  }, [dispatch, notify, record])

  return (
    <TopToolbar
      sx={{ justifyContent: 'flex-start' }}
      className={className}
      {...sanitizeListRestProps(rest)}
    >
      <Button
        onClick={handleShuffle}
        label={translate('resources.artist.actions.shuffle')}
      >
        <ShuffleIcon />
      </Button>
      <Button
        onClick={handleRadio}
        label={translate('resources.artist.actions.radio')}
      >
        <IoIosRadio />
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
  record: {},
}

export default ArtistActions
