import { useEffect, useRef, useState } from 'react'
import PropTypes from 'prop-types'
import clsx from 'clsx'
import { makeStyles } from '@material-ui/core/styles'
import Slider from '@material-ui/core/Slider'
import { GoMute, GoUnmute } from 'react-icons/go'

const LIGHT = '.react-jinke-music-player-main.light-theme &'
const muted = { color: 'hsla(0,0%,100%,.6)', [LIGHT]: { color: '#444' } }

// Sizes mirror the player's progress row, so both bars line up
const useStyle = makeStyles({
  row: {
    order: -1,
    flex: '0 0 100%',
    display: 'flex',
    alignItems: 'center',
    // Doubled to beat the player's `li { padding: 0 }` reset
    '&&': { padding: '10px 0 14px' },
  },
  side: {
    width: 55,
    display: 'inline-flex',
  },
  button: {
    cursor: 'pointer',
    background: 'none',
    border: 0,
    padding: 0,
    color: 'inherit',
  },
  mute: {
    marginRight: 5,
    ...muted,
  },
  level: {
    marginLeft: 5,
    justifyContent: 'flex-end',
    fontSize: 12,
    fontVariantNumeric: 'tabular-nums',
    ...muted,
  },
  slider: {
    flex: '1 1 auto',
    height: 4,
    padding: '13px 0',
    color: '#31c27c',
  },
  rail: {
    height: 4,
    opacity: 1,
    backgroundColor: 'hsla(0,0%,100%,.6)',
    [LIGHT]: { backgroundColor: '#e9e9e9' },
  },
  track: {
    height: 4,
  },
  thumb: {
    width: 14,
    height: 14,
    marginTop: -5,
    marginLeft: -7,
    border: '2px solid #fff',
  },
})

const SPACE_KEY_CODE = 32

// iPhone Safari locks audio.volume: it keeps a new value only until a later
// task reverts it, so the check must read it back asynchronously.
export const VOLUME_PROBE_MS = 100

const VolumeControl = ({ onVolumeChange, onMutedChange }) => {
  const classes = useStyle()
  const [adjustable, setAdjustable] = useState(true)
  const [volume, setVolume] = useState(1)
  const [isMuted, setMuted] = useState(false)
  const button = useRef(null)

  useEffect(() => {
    const probe = document.createElement('audio')
    probe.volume = 0.5
    const timer = setTimeout(
      () => setAdjustable(probe.volume === 0.5),
      VOLUME_PROBE_MS,
    )
    return () => clearTimeout(timer)
  }, [])

  // The player toggles play on Space with a native listener on its root, which
  // a React handler cannot stop in time.
  useEffect(() => {
    const el = button.current
    const stopSpace = (e) => e.keyCode === SPACE_KEY_CODE && e.stopPropagation()
    el.addEventListener('keydown', stopSpace)
    return () => el.removeEventListener('keydown', stopSpace)
  }, [])

  const applyMuted = (value) => {
    setMuted(value)
    onMutedChange(value)
  }

  const handleChange = (_, value) => {
    setVolume(value)
    onVolumeChange(value)
    if (isMuted) applyMuted(false)
  }

  const label = isMuted ? 'Unmute' : 'Mute'
  const level = isMuted ? 0 : volume

  const muteButton = (
    <button
      ref={button}
      type="button"
      className={clsx(
        classes.button,
        adjustable && [classes.side, classes.mute],
      )}
      onClick={() => applyMuted(!isMuted)}
      title={label}
      aria-label={label}
    >
      {level === 0 ? <GoMute /> : <GoUnmute />}
    </button>
  )

  if (!adjustable) {
    return <li className="item">{muteButton}</li>
  }

  return (
    <li className={classes.row}>
      {muteButton}
      <Slider
        classes={{
          root: classes.slider,
          rail: classes.rail,
          track: classes.track,
          thumb: classes.thumb,
        }}
        value={level}
        onChange={handleChange}
        min={0}
        max={1}
        step={0.01}
        aria-label="Volume"
      />
      <span className={clsx(classes.side, classes.level)} aria-hidden="true">
        {Math.round(level * 100)}%
      </span>
    </li>
  )
}

VolumeControl.propTypes = {
  onVolumeChange: PropTypes.func.isRequired,
  onMutedChange: PropTypes.func.isRequired,
}

export default VolumeControl
