/**
 * The live journey view: artwork, phase arc, progress, the next track with
 * its reason, and the steering controls. Playback itself is done by the
 * existing audio player — this component only observes and steers.
 */
import { useMemo } from 'react'
import { useSelector } from 'react-redux'
import { Button, IconButton, Typography } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import StopIcon from '@material-ui/icons/Stop'
import { Artwork } from '../../common/Artwork'
import StageStepper from './StageStepper'
import JourneyProgress from './JourneyProgress'
import JourneyControls from './JourneyControls'
import {
  computeJourneyPosition,
  dominantRemainingGenre,
} from '../engine/progress'

const useStyles = makeStyles((theme) => ({
  root: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(2),
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(2),
  },
  cover: {
    width: 96,
    height: 96,
    borderRadius: 12,
    flexShrink: 0,
    [theme.breakpoints.up('sm')]: { width: 128, height: 128 },
  },
  titleBlock: { flex: 1, minWidth: 0 },
  eyebrow: {
    fontSize: 11,
    letterSpacing: 1.5,
    textTransform: 'uppercase',
    color: theme.palette.secondary.main,
    fontWeight: 700,
  },
  journeyTitle: {
    fontWeight: 700,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  meta: { color: theme.palette.text.secondary, fontSize: 13 },
  trackTitle: {
    marginTop: theme.spacing(0.5),
    fontWeight: 600,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  reason: {
    color: theme.palette.text.secondary,
    fontSize: 13,
    fontStyle: 'italic',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    display: '-webkit-box',
    WebkitLineClamp: 2,
    WebkitBoxOrient: 'vertical',
  },
  footer: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: theme.spacing(2),
  },
}))

/**
 * @param {Object} props
 * @param {import('../engine/types').MusicJourney} props.journey
 * @param {Function} props.onAdjust adjustment handler from useJourney
 * @param {Function} props.onStop
 * @param {boolean} [props.adjusting] disables controls while rebuilding
 */
export const JourneyNowPlaying = ({ journey, onAdjust, onStop, adjusting }) => {
  const classes = useStyles()
  const queue = useSelector((state) => state.player?.queue || [])
  const current = useSelector((state) => state.player?.current || {})

  const pos = useMemo(
    () => computeJourneyPosition(journey, queue, current),
    [journey, queue, current],
  )
  const isPlaying = Boolean(current?.uuid) && current?.playing !== false

  const dominantGenre = useMemo(
    () => dominantRemainingGenre(journey, Math.max(0, pos.currentIdx + 1)),
    [journey, pos.currentIdx],
  )

  return (
    <div className={classes.root}>
      <div className={classes.header}>
        {pos.current && (
          <Artwork
            record={current?.song || pos.current.track}
            square
            className={classes.cover}
            title={pos.current.track.title}
          />
        )}
        <div className={classes.titleBlock}>
          <div className={classes.eyebrow}>Now playing · Music Journey</div>
          <Typography variant="h6" className={classes.journeyTitle}>
            {journey.title}
          </Typography>
          <div className={classes.meta}>
            {journey.narrative.map((s) => s.name).join(' → ')} ·{' '}
            {Math.round(journey.estimatedDuration / 60)} min
          </div>
          {pos.current && (
            <>
              <div className={classes.trackTitle}>
                {pos.current.track.title} — {pos.current.track.artist}
              </div>
              {pos.current.reason && (
                <div className={classes.reason}>“{pos.current.reason}”</div>
              )}
            </>
          )}
        </div>
        <IconButton
          onClick={onStop}
          aria-label="Stop the journey and clear the queue"
          size="small"
        >
          <StopIcon />
        </IconButton>
      </div>

      <StageStepper narrative={journey.narrative} currentIndex={pos.phaseIdx} />

      <JourneyProgress
        journey={journey}
        queue={queue}
        current={current}
        isPlaying={isPlaying}
      />

      <div className={classes.footer}>
        <JourneyControls
          onAdjust={onAdjust}
          dominantGenre={dominantGenre}
          disabled={adjusting}
        />
      </div>

      {journey.truncated && (
        <Button disabled size="small" color="default">
          Not enough matching tracks — the journey was shortened.
        </Button>
      )}
    </div>
  )
}

export default JourneyNowPlaying
