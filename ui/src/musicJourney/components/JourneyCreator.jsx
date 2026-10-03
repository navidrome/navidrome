/**
 * Journey Creator — mood, duration, intensity, discovery, optional genre
 * and era. Mobile-first: single column, large touch targets, everything
 * reachable with one hand.
 */
import { useMemo, useState } from 'react'
import { useSelector } from 'react-redux'
import {
  Button,
  Chip,
  CircularProgress,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Slider,
  Typography,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import NightsStayIcon from '@material-ui/icons/NightsStay'
import VisibilityIcon from '@material-ui/icons/Visibility'
import WhatshotIcon from '@material-ui/icons/Whatshot'
import WbSunnyIcon from '@material-ui/icons/WbSunny'
import WavesIcon from '@material-ui/icons/Waves'
import ExploreIcon from '@material-ui/icons/Explore'
import { MOODS } from '../engine/types'
import { buildNarrative } from '../engine/phases'

const MOOD_META = {
  chill: { icon: NightsStayIcon },
  focus: { icon: VisibilityIcon },
  energy: { icon: WhatshotIcon },
  happy: { icon: WbSunnyIcon },
  melancholic: { icon: WavesIcon },
}

const DECADES = ['1950', '1960', '1970', '1980', '1990', '2000', '2010', '2020']

const useStyles = makeStyles((theme) => ({
  root: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(3),
  },
  sectionLabel: {
    fontSize: 12,
    letterSpacing: 1.2,
    textTransform: 'uppercase',
    color: theme.palette.text.hint,
    marginBottom: theme.spacing(1),
    fontWeight: 600,
  },
  moods: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(1),
  },
  moodChip: {
    borderRadius: 22,
    padding: theme.spacing(0, 1),
    height: 40,
    fontSize: 14,
    fontWeight: 500,
  },
  sliderLabels: {
    display: 'flex',
    justifyContent: 'space-between',
    color: theme.palette.text.hint,
    fontSize: 12,
    marginTop: -theme.spacing(0.5),
  },
  selects: {
    display: 'flex',
    gap: theme.spacing(2),
    flexWrap: 'wrap',
    '& > *': { flex: '1 1 160px' },
  },
  footer: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'stretch',
    gap: theme.spacing(1.5),
    marginTop: theme.spacing(1),
  },
  arcPreview: {
    textAlign: 'center',
    color: theme.palette.text.secondary,
    fontSize: 13,
  },
  startButton: {
    borderRadius: 28,
    padding: theme.spacing(1.5),
    fontSize: 16,
    fontWeight: 700,
    letterSpacing: 1,
    background: `linear-gradient(90deg, ${theme.palette.secondary.main}, ${theme.palette.primary.main})`,
    color: theme.palette.primary.contrastText,
    '&:hover': {
      background: `linear-gradient(90deg, ${theme.palette.secondary.dark}, ${theme.palette.primary.dark})`,
    },
  },
}))

/**
 * @param {Object} props
 * @param {Array<{id: string, name: string}>} props.genres
 * @param {Function} props.onStart (preferences) => void
 * @param {boolean} props.generating
 * @param {Object} [props.initialPreferences] prefill (e.g. from history "Regenerate")
 */
export const JourneyCreator = ({
  genres,
  onStart,
  generating,
  initialPreferences,
}) => {
  const classes = useStyles()
  const stored = useSelector((state) => state.journey?.preferences)
  const base = initialPreferences || stored || {}

  const [mood, setMood] = useState(base.mood || 'chill')
  const [duration, setDuration] = useState(base.durationMinutes || 30)
  const [intensity, setIntensity] = useState(
    Math.round((base.intensity ?? 0.4) * 100),
  )
  const [discovery, setDiscovery] = useState(
    Math.round((base.discovery ?? 0.3) * 100),
  )
  const [genreId, setGenreId] = useState('')
  const [decade, setDecade] = useState(base.decade || '')

  const genre = genres.find((g) => g.id === genreId)

  const preferences = useMemo(
    () => ({
      mood,
      durationMinutes: duration,
      intensity: intensity / 100,
      discovery: discovery / 100,
      genreId: genreId || undefined,
      genre: genre?.name,
      decade: decade || undefined,
    }),
    [mood, duration, intensity, discovery, genreId, genre, decade],
  )

  const arcPreview = useMemo(() => {
    try {
      return buildNarrative(preferences)
        .map((p) => p.name)
        .join(' → ')
    } catch {
      return ''
    }
  }, [preferences])

  const handleStart = () => onStart(preferences)

  return (
    <div className={classes.root}>
      <div>
        <div className={classes.sectionLabel}>Mood</div>
        <div className={classes.moods} role="radiogroup" aria-label="Mood">
          {MOODS.map((m) => {
            const Icon = MOOD_META[m]?.icon
            const selected = mood === m
            return (
              <Chip
                key={m}
                role="radio"
                aria-checked={selected}
                icon={Icon ? <Icon /> : undefined}
                label={m.charAt(0).toUpperCase() + m.slice(1)}
                color={selected ? 'secondary' : 'default'}
                variant={selected ? 'default' : 'outlined'}
                className={classes.moodChip}
                onClick={() => setMood(m)}
              />
            )
          })}
        </div>
      </div>

      <div>
        <div className={classes.sectionLabel}>Duration</div>
        <Slider
          value={duration}
          min={10}
          max={120}
          step={5}
          marks={[
            { value: 15, label: '15' },
            { value: 30, label: '30' },
            { value: 60, label: '60' },
            { value: 90, label: '90' },
          ]}
          onChange={(_, v) => setDuration(v)}
          valueLabelDisplay="auto"
          valueLabelFormat={(v) => `${v} min`}
          aria-label="Journey duration in minutes"
        />
      </div>

      <div>
        <div className={classes.sectionLabel}>Intensity</div>
        <Slider
          value={intensity}
          min={0}
          max={100}
          onChange={(_, v) => setIntensity(v)}
          valueLabelDisplay="auto"
          valueLabelFormat={(v) => `${v}%`}
          aria-label="Journey intensity"
        />
        <div className={classes.sliderLabels}>
          <span>Calm</span>
          <span>High</span>
        </div>
      </div>

      <div>
        <div className={classes.sectionLabel}>Discovery</div>
        <Slider
          value={discovery}
          min={0}
          max={100}
          onChange={(_, v) => setDiscovery(v)}
          valueLabelDisplay="auto"
          valueLabelFormat={(v) => `${v}%`}
          aria-label="How surprising the journey should be"
        />
        <div className={classes.sliderLabels}>
          <span>Familiar</span>
          <span>Surprise</span>
        </div>
      </div>

      <div className={classes.selects}>
        <FormControl variant="outlined" size="small">
          <InputLabel id="journey-genre-label">Genre</InputLabel>
          <Select
            labelId="journey-genre-label"
            label="Genre"
            value={genreId}
            onChange={(e) => setGenreId(e.target.value)}
            aria-label="Optional genre filter"
          >
            <MenuItem value="">
              <em>Any genre</em>
            </MenuItem>
            {genres.map((g) => (
              <MenuItem key={g.id} value={g.id}>
                {g.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl variant="outlined" size="small">
          <InputLabel id="journey-decade-label">Era</InputLabel>
          <Select
            labelId="journey-decade-label"
            label="Era"
            value={decade}
            onChange={(e) => setDecade(e.target.value)}
            aria-label="Optional era filter"
          >
            <MenuItem value="">
              <em>Any era</em>
            </MenuItem>
            {DECADES.map((d) => (
              <MenuItem key={d} value={d}>
                {d}s
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </div>

      <div className={classes.footer}>
        {arcPreview && (
          <Typography className={classes.arcPreview} aria-hidden="true">
            {arcPreview}
          </Typography>
        )}
        <Button
          className={classes.startButton}
          variant="contained"
          size="large"
          disabled={generating}
          onClick={handleStart}
          startIcon={
            generating ? (
              <CircularProgress size={20} color="inherit" />
            ) : (
              <ExploreIcon />
            )
          }
          aria-label="Start the journey"
        >
          {generating ? 'Preparing your journey…' : 'Start Journey'}
        </Button>
      </div>
    </div>
  )
}

export default JourneyCreator
