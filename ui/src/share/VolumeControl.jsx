import { useCallback, useEffect, useState } from 'react'
import { makeStyles } from '@material-ui/core/styles'
import Slider from '@material-ui/core/Slider'
import VolumeUpIcon from '@material-ui/icons/VolumeUp'
import VolumeOffIcon from '@material-ui/icons/VolumeOff'

const useStyle = makeStyles({
  root: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    // Touch devices have hardware volume buttons (and iOS ignores
    // `audio.volume` anyway), so only show the control where a pointer exists.
    '@media (hover: none)': {
      display: 'none',
    },
  },
  button: {
    display: 'inline-flex',
    cursor: 'pointer',
    background: 'none',
    border: 0,
    padding: 0,
    color: 'inherit',
  },
  slider: {
    width: 100,
    marginLeft: 12,
    color: '#31c27c',
  },
})

// The player maps its volume bar to the real volume with a square, so that the
// bar feels linear. Use the same mapping, as the main player does.
const toBarValue = (volume) => Math.sqrt(volume)
const toVolume = (barValue) => barValue * barValue

// Volume slider for the share page. The share page always uses the player's
// "mobile" layout, which does not include a volume control.
const VolumeControl = ({ audio }) => {
  const classes = useStyle()
  const [value, setValue] = useState(1)
  const [muted, setMuted] = useState(false)

  useEffect(() => {
    if (!audio) return
    const sync = () => {
      setValue(toBarValue(audio.volume))
      setMuted(audio.muted)
    }
    sync()
    audio.addEventListener('volumechange', sync)
    return () => audio.removeEventListener('volumechange', sync)
  }, [audio])

  const handleChange = useCallback(
    (_, newValue) => {
      if (!audio) return
      audio.muted = false
      audio.volume = toVolume(newValue)
    },
    [audio],
  )

  const toggleMute = useCallback(() => {
    if (audio) audio.muted = !audio.muted
  }, [audio])

  const silent = muted || value === 0

  return (
    <li className={`item ${classes.root}`}>
      <button
        type="button"
        className={classes.button}
        onClick={toggleMute}
        title={muted ? 'Unmute' : 'Mute'}
        aria-label={muted ? 'Unmute' : 'Mute'}
      >
        {silent ? <VolumeOffIcon /> : <VolumeUpIcon />}
      </button>
      <Slider
        className={classes.slider}
        value={muted ? 0 : value}
        onChange={handleChange}
        min={0}
        max={1}
        step={0.01}
        disabled={!audio}
        aria-label="Volume"
      />
    </li>
  )
}

export default VolumeControl
