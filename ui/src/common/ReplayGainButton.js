import IconButton from '@material-ui/core/IconButton'
import AlbumIcon from '@material-ui/icons/Album';
import AudiotrackIcon from '@material-ui/icons/Audiotrack';
import PropTypes from 'prop-types'
import { useCallback } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { toggleGain } from '../actions';

export const ReplayGainButton = ({
  size,
  component: Button,
}) => {
  const dispatch = useDispatch();
  const isAlbumGain = useSelector((state) => state.player?.isAlbumGain)


  const handleToggleGainMode = useCallback(
    (e) => {
      e.preventDefault()
      dispatch(toggleGain())
      e.stopPropagation()
    },
    [dispatch]
  )

  return (
    <Button 
      onClick={handleToggleGainMode}
      size="small"
    >
      {
        isAlbumGain ? 
          <AlbumIcon fontSize={size} /> :
          <AudiotrackIcon fontSize={size} />
      }
    </Button>
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