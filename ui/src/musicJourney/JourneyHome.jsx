/**
 * Music Journey hub — the entry point of the feature.
 *
 * Shows either the live journey view (while one is playing) or the
 * creation experience: creator, a "tonight's journey" suggestion built
 * from the user's last preferences, and the journey history.
 */
import { useEffect, useMemo, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Title, useDataProvider, useNotify, useTranslate } from 'react-admin'
import { Button, Card, CardContent, Grid, Typography } from '@material-ui/core'
import { Alert } from '@material-ui/lab'
import { makeStyles } from '@material-ui/core/styles'
import ExploreIcon from '@material-ui/icons/Explore'
import NightsStayIcon from '@material-ui/icons/NightsStay'
import WbSunnyIcon from '@material-ui/icons/WbSunny'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import JourneyCreator from './components/JourneyCreator'
import JourneyNowPlaying from './components/JourneyNowPlaying'
import JourneyHistory from './components/JourneyHistory'
import { useJourney } from './hooks/useJourney'
import { generateJourneyName } from './engine/nameGenerator'
import { buildNarrative } from './engine/phases'
import {
  deleteJourneyHistoryEntry,
  dismissJourneyError,
  journeyEnded,
  setJourneyPreferences,
  toggleJourneyLike,
} from './state/journeyActions'

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(2),
    maxWidth: 1100,
    margin: '0 auto',
    [theme.breakpoints.down('xs')]: { padding: theme.spacing(1) },
  },
  hero: {
    borderRadius: 16,
    padding: theme.spacing(3),
    marginBottom: theme.spacing(3),
    color: '#fff',
    background: `linear-gradient(120deg, ${theme.palette.primary.main}, ${theme.palette.secondary.main})`,
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(2),
  },
  heroIcon: { opacity: 0.9 },
  heroTitle: {
    fontWeight: 800,
    letterSpacing: 2,
    textTransform: 'uppercase',
  },
  heroTagline: { opacity: 0.85 },
  card: { height: '100%' },
  cardTitle: {
    fontWeight: 700,
    marginBottom: theme.spacing(2),
  },
  suggestionCard: {
    marginBottom: theme.spacing(3),
    borderRadius: 12,
  },
  suggestionEyebrow: {
    fontSize: 11,
    letterSpacing: 1.5,
    textTransform: 'uppercase',
    color: theme.palette.text.hint,
    fontWeight: 700,
  },
  suggestionTitle: { fontWeight: 700 },
  suggestionMeta: {
    color: theme.palette.text.secondary,
    marginTop: theme.spacing(0.5),
    marginBottom: theme.spacing(1.5),
    fontSize: 14,
  },
  error: { marginBottom: theme.spacing(2) },
}))

const timeOfDayKey = () => {
  const h = new Date().getHours()
  if (h < 6) return 'night'
  if (h < 12) return 'morning'
  if (h < 18) return 'afternoon'
  return 'evening'
}

const errorCopy = (error, prefs) => {
  if (!error) return null
  switch (error.code) {
    case 'EMPTY_LIBRARY':
      return {
        title: 'Unable to generate journey',
        message:
          'Your library looks empty right now. Check that your music server is reachable and has finished scanning.',
        hints: [],
      }
    case 'NOT_ENOUGH_TRACKS':
      return {
        title: 'Unable to generate journey',
        message: "Your library doesn't contain enough matching tracks.",
        hints: [
          'increasing discovery',
          'choosing another mood',
          'shortening the journey',
          ...(prefs?.genre || prefs?.decade
            ? ['clearing the genre/era filter']
            : []),
        ],
      }
    default:
      return {
        title: 'Unable to generate journey',
        message:
          'Something went wrong while talking to the server. Please try again.',
        hints: [],
      }
  }
}

export const JourneyHome = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const dispatch = useDispatch()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const { generate, adjust, stop, replay } = useJourney()

  const journeyState = useSelector((state) => state.journey)
  const playerQueue = useSelector((state) => state.player?.queue || [])
  const active = journeyState?.active
  const [adjusting, setAdjusting] = useState(false)
  const [genres, setGenres] = useState([])
  const [prefill, setPrefill] = useState(null)

  // Genre list for the optional filter (reuses the existing genre resource).
  useEffect(() => {
    let cancelled = false
    dataProvider
      .getList('genre', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'name', order: 'ASC' },
        filter: {},
      })
      .then((res) => {
        if (!cancelled) setGenres(res.data || [])
      })
      .catch(() => {
        // Genre filter is optional; an error here only hides the select.
      })
    return () => {
      cancelled = true
    }
  }, [dataProvider])

  // A journey whose queue disappeared was ended elsewhere; keep UI honest.
  const livePosition = useMemo(() => {
    if (!active) return null
    const ids = new Set(active.journey.tracks.map((t) => t.track.id))
    return playerQueue.some((item) => ids.has(item?.trackId)) ? 'live' : 'gone'
  }, [active, playerQueue])

  useEffect(() => {
    if (active && livePosition === 'gone') {
      dispatch(journeyEnded('interrupted'))
    }
  }, [active, livePosition, dispatch])

  const suggestion = useMemo(() => {
    const prefs = journeyState?.preferences
    if (!prefs) return null
    const prefsWithSeed = { ...prefs, seed: `${new Date().toDateString()}` }
    try {
      return {
        prefs: prefsWithSeed,
        title: generateJourneyName(prefsWithSeed),
        arc: buildNarrative(prefsWithSeed)
          .map((p) => p.name)
          .join(' → '),
      }
    } catch {
      return null
    }
  }, [journeyState?.preferences])

  const lastEntry = journeyState?.history?.[0]

  const handleStart = async (prefs) => {
    const journey = await generate(prefs)
    if (journey) {
      notify(`Journey "${journey.title}" started`, 'info')
    }
  }

  const handleAdjust = async (kind) => {
    setAdjusting(true)
    try {
      const rebuilt = await adjust(kind)
      if (rebuilt) {
        notify('Journey re-routed for the remaining tracks', 'info')
      }
    } finally {
      setAdjusting(false)
    }
  }

  const handleReplay = async (entry) => {
    const ok = await replay(entry)
    if (ok) {
      dispatch(
        setJourneyPreferences({
          mood: entry.mood,
          durationMinutes: entry.durationMinutes,
          intensity: entry.intensity,
          discovery: entry.discovery,
          genre: entry.genre,
          decade: entry.decade,
        }),
      )
      notify(`Playing "${entry.title}" again`, 'info')
    } else {
      notify('Those tracks are no longer available in your library', 'warning')
    }
  }

  const handleRegenerate = (entry) => {
    setPrefill({
      mood: entry.mood,
      durationMinutes: entry.durationMinutes,
      intensity: entry.intensity,
      discovery: entry.discovery,
      decade: entry.decade,
    })
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const error = errorCopy(journeyState?.error, journeyState?.preferences)
  const isLive = active && livePosition === 'live'
  const tod = timeOfDayKey()

  return (
    <div className={classes.root}>
      <Title title={translate('journey.name', { _: 'Music Journey' })} />
      <div className={classes.hero}>
        <ExploreIcon className={classes.heroIcon} style={{ fontSize: 48 }} />
        <div>
          <Typography variant="h5" className={classes.heroTitle}>
            {translate('journey.name', { _: 'Music Journey' })}
          </Typography>
          <div className={classes.heroTagline}>
            {translate('journey.tagline', {
              _: 'Your library, arranged as a journey — not a shuffle.',
            })}
          </div>
        </div>
      </div>

      {error && (
        <Alert
          severity="error"
          className={classes.error}
          onClose={() => dispatch(dismissJourneyError())}
          role="alert"
        >
          <strong>{error.title}.</strong> {error.message}
          {error.hints.length > 0 && (
            <>
              <br />
              Try:{' '}
              {error.hints.map((h, i) => (
                <span key={h}>
                  {i > 0 && ' · '}• {h}
                </span>
              ))}
            </>
          )}
        </Alert>
      )}

      {isLive ? (
        <Card variant="outlined">
          <CardContent>
            <JourneyNowPlaying
              journey={active.journey}
              onAdjust={handleAdjust}
              onStop={stop}
              adjusting={adjusting || journeyState.generating}
            />
          </CardContent>
        </Card>
      ) : (
        <Grid container spacing={3}>
          <Grid item xs={12} md={7}>
            <Card variant="outlined" className={classes.card}>
              <CardContent>
                <Typography variant="h6" className={classes.cardTitle}>
                  {translate('journey.createTitle', {
                    _: 'What kind of journey do you want?',
                  })}
                </Typography>
                <JourneyCreator
                  key={prefill ? JSON.stringify(prefill) : 'default'}
                  initialPreferences={prefill}
                  genres={genres}
                  onStart={handleStart}
                  generating={journeyState.generating}
                />
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={5}>
            {suggestion && (
              <Card variant="outlined" className={classes.suggestionCard}>
                <CardContent>
                  <div className={classes.suggestionEyebrow}>
                    {tod === 'evening' || tod === 'night'
                      ? "Tonight's Journey"
                      : "Today's Journey"}
                    {tod === 'morning' && (
                      <WbSunnyIcon
                        fontSize="small"
                        style={{ marginLeft: 6, verticalAlign: 'middle' }}
                      />
                    )}
                    {(tod === 'evening' || tod === 'night') && (
                      <NightsStayIcon
                        fontSize="small"
                        style={{ marginLeft: 6, verticalAlign: 'middle' }}
                      />
                    )}
                  </div>
                  <Typography variant="h6" className={classes.suggestionTitle}>
                    “{suggestion.title}”
                  </Typography>
                  <div className={classes.suggestionMeta}>
                    {suggestion.prefs.durationMinutes} min · {suggestion.arc}
                  </div>
                  <Button
                    variant="contained"
                    color="secondary"
                    startIcon={<PlayArrowIcon />}
                    onClick={() => handleStart(suggestion.prefs)}
                    disabled={journeyState.generating}
                    aria-label="Start the suggested journey"
                  >
                    Start
                  </Button>
                  {lastEntry && (
                    <Button
                      style={{ marginLeft: 8 }}
                      onClick={() => handleReplay(lastEntry)}
                      aria-label={`Continue the last journey: ${lastEntry.title}`}
                    >
                      Continue “{lastEntry.title}”
                    </Button>
                  )}
                </CardContent>
              </Card>
            )}
          </Grid>
        </Grid>
      )}

      <JourneyHistory
        entries={journeyState?.history || []}
        onReplay={handleReplay}
        onRegenerate={handleRegenerate}
        onDelete={(id) => dispatch(deleteJourneyHistoryEntry(id))}
        onToggleLike={(id) => dispatch(toggleJourneyLike(id))}
      />
    </div>
  )
}

export default JourneyHome
