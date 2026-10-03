/**
 * Live journey progress: elapsed/remaining time, current phase and the next
 * track with the engine's explanation for it.
 *
 * The player only pushes `currentTime` into Redux on play/pause/track
 * events, so this component adds a 1s ticker while playing to keep the bar
 * moving between those events.
 */
import { useMemo, useState } from 'react'
import { makeStyles } from '@material-ui/core/styles'
import Typography from '@material-ui/core/Typography'
import { useInterval } from '../../common'
import { computeJourneyPosition } from '../engine/progress'

const formatDuration = (seconds) => {
  const s = Math.max(0, Math.round(seconds))
  const m = Math.floor(s / 60)
  const rest = s % 60
  return `${m}:${rest.toString().padStart(2, '0')}`
}

const useStyles = makeStyles((theme) => ({
  root: { width: '100%' },
  barOuter: {
    position: 'relative',
    height: 8,
    borderRadius: 4,
    background: theme.palette.divider,
    overflow: 'hidden',
  },
  barInner: {
    position: 'absolute',
    top: 0,
    left: 0,
    bottom: 0,
    borderRadius: 4,
    background: `linear-gradient(90deg, ${theme.palette.secondary.main}, ${theme.palette.primary.main})`,
    transition: 'width 1s linear',
    '@media (prefers-reduced-motion: reduce)': { transition: 'none' },
  },
  times: {
    display: 'flex',
    justifyContent: 'space-between',
    marginTop: theme.spacing(0.5),
    color: theme.palette.text.secondary,
    fontSize: 12,
  },
  next: {
    marginTop: theme.spacing(2),
    padding: theme.spacing(1.5),
    borderRadius: 10,
    background:
      theme.palette.type === 'dark'
        ? 'rgba(255,255,255,0.04)'
        : 'rgba(0,0,0,0.03)',
  },
  nextLabel: {
    fontSize: 11,
    letterSpacing: 1,
    textTransform: 'uppercase',
    color: theme.palette.text.hint,
    marginBottom: 4,
  },
  reason: {
    marginTop: theme.spacing(0.5),
    color: theme.palette.text.secondary,
    fontSize: 13,
    fontStyle: 'italic',
  },
}))

/**
 * @param {Object} props
 * @param {import('../engine/types').MusicJourney} props.journey
 * @param {Array} props.queue player queue
 * @param {Object} props.current current player info (Redux state.player.current)
 * @param {boolean} props.isPlaying
 */
export const JourneyProgress = ({ journey, queue, current, isPlaying }) => {
  const classes = useStyles()
  const [, setTick] = useState(0)

  // Tick once per second while playing so elapsed time stays alive between
  // the player's Redux updates.
  useInterval(() => setTick((t) => t + 1), isPlaying ? 1000 : null)

  const pos = useMemo(
    () => computeJourneyPosition(journey, queue, current),
    [journey, queue, current],
  )

  const pct = pos.total > 0 ? Math.min(100, (pos.elapsed / pos.total) * 100) : 0

  return (
    <div className={classes.root}>
      <div
        className={classes.barOuter}
        role="progressbar"
        aria-valuenow={Math.round(pct)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label="Journey progress"
      >
        <div className={classes.barInner} style={{ width: `${pct}%` }} />
      </div>
      <div className={classes.times}>
        <span>{formatDuration(pos.elapsed)}</span>
        <span>-{formatDuration(pos.total - pos.elapsed)} remaining</span>
      </div>
      {pos.next && (
        <div className={classes.next} aria-live="polite">
          <div className={classes.nextLabel}>Next up</div>
          <Typography variant="body2">
            <strong>{pos.next.track.title}</strong> — {pos.next.track.artist}
            <span aria-hidden="true"> · </span>
            {pos.next.stage?.name}
          </Typography>
          {pos.next.reason && (
            <div className={classes.reason}>“{pos.next.reason}”</div>
          )}
        </div>
      )}
    </div>
  )
}

export default JourneyProgress
