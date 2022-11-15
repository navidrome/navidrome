import { Tooltip } from '@material-ui/core'
import IconButton from '@material-ui/core/IconButton'
import AlbumIcon from '@material-ui/icons/Album'
import AudiotrackIcon from '@material-ui/icons/Audiotrack'
import NotInterestedIcon from '@material-ui/icons/NotInterested'
import PropTypes from 'prop-types'
import { useCallback } from 'react'
import { useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { toggleGain } from '../actions'

export const ReplayGainButton = ({ size, component: Button }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const gainMode = useSelector((state) => state.player?.gainMode)

  const handleToggleGainMode = useCallback(
    (e) => {
      e.preventDefault()
      dispatch(toggleGain())
      e.stopPropagation()
    },
    [dispatch]
  )

  return (
    <Tooltip title={translate(`player.gain.${gainMode ?? 'none'}`)}>
      <Button onClick={handleToggleGainMode} size="small">
        {gainMode === 'album' ? (
          <AlbumIcon fontSize={size} />
        ) : gainMode === 'track' ? (
          <AudiotrackIcon fontSize={size} />
        ) : (
          <NotInterestedIcon fontSize={size} />
        )}
      </Button>
    </Tooltip>
  )
}

ReplayGainButton.propTypes = {
  size: PropTypes.string,
  component: PropTypes.object,
}

ReplayGainButton.defaultProps = {
  size: 'small',
  component: IconButton,
}
