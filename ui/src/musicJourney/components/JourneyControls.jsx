/**
 * Live journey steering. Each chip adjusts the remaining part of the
 * journey — never what was already played.
 */
import { Chip, makeStyles } from '@material-ui/core'
import WhatshotIcon from '@material-ui/icons/Whatshot'
import SpaIcon from '@material-ui/icons/Spa'
import FavoriteIcon from '@material-ui/icons/Favorite'
import CasinoIcon from '@material-ui/icons/Casino'
import NotInterestedIcon from '@material-ui/icons/NotInterested'

const useStyles = makeStyles((theme) => ({
  root: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(1),
    marginTop: theme.spacing(2),
  },
  chip: {
    borderRadius: 20,
    fontWeight: 500,
  },
}))

/**
 * @param {Object} props
 * @param {(kind: string) => void} props.onAdjust
 * @param {string|null} props.dominantGenre most frequent genre in the
 *   remaining tracks; when present a "Less <genre>" chip is offered
 * @param {boolean} [props.disabled]
 */
export const JourneyControls = ({ onAdjust, dominantGenre, disabled }) => {
  const classes = useStyles()

  const controls = [
    {
      kind: 'more-energy',
      label: 'More energetic',
      icon: <WhatshotIcon fontSize="small" />,
    },
    {
      kind: 'more-relaxed',
      label: 'More relaxed',
      icon: <SpaIcon fontSize="small" />,
    },
    {
      kind: 'more-familiar',
      label: 'More familiar',
      icon: <FavoriteIcon fontSize="small" />,
    },
    {
      kind: 'surprise',
      label: 'Surprise me',
      icon: <CasinoIcon fontSize="small" />,
    },
  ]
  if (dominantGenre) {
    controls.push({
      kind: 'less-genre',
      label: `Less ${dominantGenre}`,
      icon: <NotInterestedIcon fontSize="small" />,
    })
  }

  return (
    <div className={classes.root} aria-label="Adjust the journey direction">
      {controls.map((c) => (
        <Chip
          key={c.kind}
          className={classes.chip}
          icon={c.icon}
          label={c.label}
          color={c.kind === 'surprise' ? 'secondary' : 'default'}
          variant={c.kind === 'surprise' ? 'default' : 'outlined'}
          disabled={disabled}
          onClick={() => onAdjust(c.kind)}
          aria-label={c.label}
        />
      ))}
    </div>
  )
}

export default JourneyControls
