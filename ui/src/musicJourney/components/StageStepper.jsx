/**
 * Visual arc of the journey: one node per narrative phase.
 * Completed phases get a check, the current one pulses, future ones stay
 * outlined. Purely presentational and keyboard/screen-reader friendly.
 */
import { makeStyles } from '@material-ui/core/styles'
import CheckIcon from '@material-ui/icons/Check'
import clsx from 'clsx'

const useStyles = makeStyles((theme) => ({
  root: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: theme.spacing(1),
    overflowX: 'auto',
    padding: theme.spacing(0.5, 0),
  },
  stage: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    minWidth: 64,
    flex: '1 1 0',
  },
  dotRow: {
    display: 'flex',
    alignItems: 'center',
    width: '100%',
  },
  line: {
    flex: 1,
    height: 2,
    background: theme.palette.divider,
  },
  lineDone: {
    background: theme.palette.secondary.main,
  },
  dot: {
    width: 26,
    height: 26,
    margin: '0 auto',
    borderRadius: '50%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    border: `2px solid ${theme.palette.divider}`,
    color: theme.palette.text.hint,
    background: 'transparent',
    fontSize: 13,
    flexShrink: 0,
  },
  dotDone: {
    borderColor: theme.palette.secondary.main,
    background: theme.palette.secondary.main,
    color: theme.palette.secondary.contrastText,
  },
  dotCurrent: {
    borderColor: theme.palette.secondary.main,
    color: theme.palette.secondary.main,
    animation: '$journeyPulse 2s ease-in-out infinite',
    '@media (prefers-reduced-motion: reduce)': { animation: 'none' },
  },
  '@keyframes journeyPulse': {
    '0%': { boxShadow: `0 0 0 0 ${theme.palette.secondary.main}44` },
    '50%': { boxShadow: `0 0 0 8px transparent` },
    '100%': { boxShadow: `0 0 0 0 transparent` },
  },
  label: {
    marginTop: theme.spacing(0.5),
    fontSize: 12,
    textAlign: 'center',
    color: theme.palette.text.secondary,
    whiteSpace: 'nowrap',
  },
  labelCurrent: {
    color: theme.palette.text.primary,
    fontWeight: 600,
  },
}))

/**
 * @param {Object} props
 * @param {Array<{id: string, name: string}>} props.narrative
 * @param {number} props.currentIndex index of the active phase
 * @param {boolean} [props.completed] renders everything as done
 */
export const StageStepper = ({
  narrative,
  currentIndex,
  completed = false,
}) => {
  const classes = useStyles()
  const activeIdx = completed ? narrative.length - 1 : currentIndex

  return (
    <div className={classes.root} role="list" aria-label="Journey phases">
      {narrative.map((stage, i) => {
        const done = completed || i < activeIdx
        const isCurrent = !completed && i === activeIdx
        return (
          <div
            key={stage.id || i}
            className={classes.stage}
            role="listitem"
            aria-current={isCurrent ? 'step' : undefined}
            aria-label={`${stage.name}: ${
              done ? 'completed' : isCurrent ? 'in progress' : 'upcoming'
            }`}
          >
            <div className={classes.dotRow}>
              {i > 0 && (
                <div className={clsx(classes.line, done && classes.lineDone)} />
              )}
              <div
                className={clsx(
                  classes.dot,
                  done && classes.dotDone,
                  isCurrent && classes.dotCurrent,
                )}
              >
                {done ? <CheckIcon fontSize="small" /> : i + 1}
              </div>
              {i < narrative.length - 1 && (
                <div
                  className={clsx(
                    classes.line,
                    (completed || i < activeIdx) && classes.lineDone,
                  )}
                />
              )}
            </div>
            <div
              className={clsx(classes.label, isCurrent && classes.labelCurrent)}
            >
              {stage.name}
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default StageStepper
