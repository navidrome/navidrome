// THROWAWAY PROTOTYPE: Which songs-first layout best balances discovery, browsing, and a prominent player?
import { useMemo } from 'react'
import {
  Avatar,
  Box,
  Chip,
  Divider,
  LinearProgress,
  Paper,
  Typography,
} from '@material-ui/core'
import { alpha, makeStyles } from '@material-ui/core/styles'
import AlbumIcon from '@material-ui/icons/Album'
import CategoryIcon from '@material-ui/icons/Category'
import LibraryMusicIcon from '@material-ui/icons/LibraryMusic'
import MoreHorizIcon from '@material-ui/icons/MoreHoriz'
import PauseRoundedIcon from '@material-ui/icons/PauseRounded'
import PlayArrowRoundedIcon from '@material-ui/icons/PlayArrowRounded'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import SearchIcon from '@material-ui/icons/Search'
import SkipNextRoundedIcon from '@material-ui/icons/SkipNextRounded'
import SkipPreviousRoundedIcon from '@material-ui/icons/SkipPreviousRounded'
import WbSunnyOutlinedIcon from '@material-ui/icons/WbSunnyOutlined'
import { useListContext } from 'react-admin'
import { useSelector } from 'react-redux'
import { useLocation } from 'react-router-dom'
import { DurationField } from '../common'
import { Artwork } from '../common/Artwork'
import { PrototypeSwitcher } from '../common/PrototypeSwitcher'

const variantNames = {
  A: 'Library rail',
  B: 'Listening room',
  C: 'Minimal console',
}

const useStyles = makeStyles((theme) => {
  const panel = alpha(theme.palette.background.paper, 0.82)
  const soft = alpha(theme.palette.text.primary, 0.06)
  const accent = theme.palette.primary.main
  return {
    root: {
      minHeight: 'calc(100vh - 124px)',
      color: theme.palette.text.primary,
      '& *': { boxSizing: 'border-box' },
    },
    eyebrow: {
      color: theme.palette.text.secondary,
      fontSize: 11,
      fontWeight: 800,
      letterSpacing: '0.14em',
      textTransform: 'uppercase',
    },
    muted: { color: theme.palette.text.secondary },
    title: { fontWeight: 800, letterSpacing: '-0.035em' },
    panel: {
      border: `1px solid ${theme.palette.divider}`,
      borderRadius: 22,
      background: panel,
      backdropFilter: 'blur(18px)',
      boxShadow: 'none',
    },
    chips: { display: 'flex', flexWrap: 'wrap', gap: theme.spacing(1) },
    chip: { background: soft, border: 0, fontWeight: 600 },
    tracks: { overflow: 'hidden' },
    row: {
      display: 'grid',
      gridTemplateColumns:
        '38px minmax(180px, 1.6fr) minmax(110px, 1fr) 54px 28px',
      alignItems: 'center',
      gap: theme.spacing(1.5),
      minHeight: 58,
      padding: theme.spacing(0.75, 1.5),
      borderBottom: `1px solid ${alpha(theme.palette.divider, 0.65)}`,
      '&:last-child': { borderBottom: 0 },
      '&:hover': { background: soft },
      [theme.breakpoints.down('sm')]: {
        gridTemplateColumns: '34px minmax(140px, 1fr) 48px',
      },
    },
    rowIndex: { textAlign: 'center', color: theme.palette.text.secondary },
    trackTitle: {
      fontWeight: 700,
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
    },
    trackMeta: {
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
    },
    hideSmall: { [theme.breakpoints.down('sm')]: { display: 'none' } },
    coverSmall: { width: 38, height: 38, borderRadius: 9, background: soft },
    playerCover: {
      width: '100%',
      aspectRatio: '1 / 1',
      borderRadius: 18,
      background: `linear-gradient(145deg, ${soft}, ${alpha(accent, 0.22)})`,
      boxShadow: `0 22px 50px ${alpha(theme.palette.common.black, 0.25)}`,
    },
    controls: {
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      gap: theme.spacing(2),
      marginTop: theme.spacing(2),
    },
    play: {
      display: 'grid',
      placeItems: 'center',
      width: 52,
      height: 52,
      borderRadius: '50%',
      color: theme.palette.getContrastText(accent),
      background: accent,
    },
    progress: { height: 4, borderRadius: 4, marginTop: theme.spacing(2.5) },
    aGrid: {
      display: 'grid',
      gridTemplateColumns: 'minmax(0, 1fr) 310px',
      gap: theme.spacing(2),
      [theme.breakpoints.down('sm')]: { gridTemplateColumns: '1fr' },
    },
    aPlayer: {
      position: 'sticky',
      top: theme.spacing(2),
      alignSelf: 'start',
      padding: theme.spacing(2.5),
      [theme.breakpoints.down('sm')]: { position: 'static', order: -1 },
    },
    aHeader: {
      display: 'flex',
      justifyContent: 'space-between',
      gap: theme.spacing(2),
      marginBottom: theme.spacing(2),
    },
    bHero: {
      position: 'relative',
      overflow: 'hidden',
      display: 'grid',
      gridTemplateColumns: '240px minmax(0, 1fr)',
      alignItems: 'end',
      gap: theme.spacing(4),
      minHeight: 330,
      padding: theme.spacing(4),
      marginBottom: theme.spacing(2),
      borderRadius: 30,
      background: `radial-gradient(circle at 18% 20%, ${alpha(accent, 0.48)}, transparent 38%), linear-gradient(125deg, ${theme.palette.background.paper}, ${alpha(accent, 0.1)})`,
      [theme.breakpoints.down('sm')]: {
        gridTemplateColumns: '1fr',
        padding: theme.spacing(2),
        minHeight: 0,
      },
    },
    bCover: {
      width: 240,
      height: 240,
      borderRadius: 26,
      [theme.breakpoints.down('sm')]: { width: 170, height: 170 },
    },
    bBelow: {
      display: 'grid',
      gridTemplateColumns: '220px minmax(0, 1fr)',
      gap: theme.spacing(2),
      [theme.breakpoints.down('sm')]: { gridTemplateColumns: '1fr' },
    },
    bDiscover: { padding: theme.spacing(2) },
    discoverItem: {
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(1.5),
      padding: theme.spacing(1.25, 0),
    },
    cRoot: {
      display: 'grid',
      gridTemplateColumns: '170px minmax(0, 1fr)',
      gap: theme.spacing(2),
      [theme.breakpoints.down('sm')]: { gridTemplateColumns: '1fr' },
    },
    cNav: { padding: theme.spacing(2), alignSelf: 'stretch' },
    cNavItem: {
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(1.25),
      padding: theme.spacing(1.25),
      borderRadius: 12,
      fontWeight: 700,
    },
    cActive: { color: accent, background: alpha(accent, 0.12) },
    cSearch: {
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(1),
      padding: theme.spacing(1.25, 1.75),
      borderRadius: 999,
      background: soft,
      color: theme.palette.text.secondary,
    },
    cDock: {
      display: 'grid',
      gridTemplateColumns: '58px minmax(0, 1fr) minmax(150px, 0.8fr) 120px',
      alignItems: 'center',
      gap: theme.spacing(2),
      padding: theme.spacing(1.5),
      marginBottom: theme.spacing(2),
      [theme.breakpoints.down('sm')]: { gridTemplateColumns: '52px 1fr 90px' },
    },
    cDockCover: { width: 58, height: 58, borderRadius: 14 },
  }
})

const valuesFor = (songs, field) => {
  const values = songs.flatMap((song) => {
    if (field === 'mood') return song.tags?.mood || []
    const value = song[field]
    return Array.isArray(value) ? value : value ? [value] : []
  })
  return [...new Set(values)].slice(0, 6)
}

const TrackRows = ({ songs, showCovers = false }) => {
  const classes = useStyles()
  return (
    <div className={classes.tracks}>
      {songs.map((song, index) => (
        <div className={classes.row} key={song.id}>
          {showCovers ? (
            <Artwork
              className={classes.coverSmall}
              record={song}
              size={80}
              title={song.title}
            />
          ) : (
            <Typography className={classes.rowIndex} variant="body2">
              {String(index + 1).padStart(2, '0')}
            </Typography>
          )}
          <div style={{ minWidth: 0 }}>
            <Typography className={classes.trackTitle} variant="body2">
              {song.title}
            </Typography>
            <Typography
              className={`${classes.muted} ${classes.trackMeta}`}
              variant="caption"
            >
              {song.artist || 'Unknown artist'}
            </Typography>
          </div>
          <Typography
            className={`${classes.muted} ${classes.trackMeta} ${classes.hideSmall}`}
            variant="body2"
          >
            {song.album || 'Single'}
          </Typography>
          <Typography className={classes.muted} variant="caption">
            <DurationField record={song} source="duration" />
          </Typography>
          <MoreHorizIcon className={classes.muted} fontSize="small" />
        </div>
      ))}
    </div>
  )
}

const PlayerPanel = ({ song, compact = false }) => {
  const classes = useStyles()
  if (!song) return null
  return (
    <>
      {!compact && (
        <Artwork
          record={song}
          size={500}
          square
          className={classes.playerCover}
          title={song.title}
        />
      )}
      <Box mt={compact ? 0 : 3} textAlign={compact ? 'left' : 'center'}>
        <Typography className={classes.eyebrow}>Now playing</Typography>
        <Typography className={classes.title} variant={compact ? 'h5' : 'h6'}>
          {song.title}
        </Typography>
        <Typography className={classes.muted} variant="body2">
          {song.artist} · {song.album}
        </Typography>
      </Box>
      <LinearProgress
        className={classes.progress}
        variant="determinate"
        value={36}
      />
      <div className={classes.controls} aria-hidden="true">
        <SkipPreviousRoundedIcon />
        <span className={classes.play}>
          <PauseRoundedIcon />
        </span>
        <SkipNextRoundedIcon />
      </div>
    </>
  )
}

const VariantA = ({ songs, current, genres, moods }) => {
  const classes = useStyles()
  return (
    <div className={classes.aGrid}>
      <main>
        <header className={classes.aHeader}>
          <div>
            <Typography className={classes.eyebrow}>Your library</Typography>
            <Typography className={classes.title} variant="h3">
              Songs
            </Typography>
          </div>
          <Typography className={classes.muted} variant="body2">
            {songs.length} shown
          </Typography>
        </header>
        <div className={classes.chips} style={{ marginBottom: 16 }}>
          {[...genres, ...moods].slice(0, 7).map((item) => (
            <Chip className={classes.chip} key={item} label={item} />
          ))}
        </div>
        <Paper className={`${classes.panel} ${classes.tracks}`}>
          <TrackRows songs={songs} showCovers />
        </Paper>
      </main>
      <Paper className={`${classes.panel} ${classes.aPlayer}`}>
        <PlayerPanel song={current} />
      </Paper>
    </div>
  )
}

const VariantB = ({ songs, current, genres, moods }) => {
  const classes = useStyles()
  return (
    <div>
      <section className={classes.bHero}>
        <Artwork
          record={current}
          size={500}
          square
          className={classes.bCover}
          title={current?.title}
        />
        <div>
          <Typography className={classes.eyebrow}>
            Continue listening
          </Typography>
          <Typography className={classes.title} variant="h2">
            {current?.title || 'Your music'}
          </Typography>
          <Typography className={classes.muted} variant="h6">
            {current?.artist} · {current?.album}
          </Typography>
          <Box display="flex" alignItems="center" mt={3} style={{ gap: 18 }}>
            <span className={classes.play}>
              <PlayArrowRoundedIcon />
            </span>
            <LinearProgress
              style={{ flex: 1, maxWidth: 360 }}
              className={classes.progress}
              variant="determinate"
              value={36}
            />
          </Box>
        </div>
      </section>
      <div className={classes.bBelow}>
        <Paper className={`${classes.panel} ${classes.bDiscover}`}>
          <Typography className={classes.eyebrow}>Explore</Typography>
          {genres.slice(0, 3).map((genre) => (
            <div className={classes.discoverItem} key={genre}>
              <Avatar>
                <CategoryIcon />
              </Avatar>
              <div>
                <Typography variant="body2">
                  <b>{genre}</b>
                </Typography>
                <Typography className={classes.muted} variant="caption">
                  Category
                </Typography>
              </div>
            </div>
          ))}
          <Divider />
          {moods.slice(0, 3).map((mood) => (
            <div className={classes.discoverItem} key={mood}>
              <Avatar>
                <WbSunnyOutlinedIcon />
              </Avatar>
              <div>
                <Typography variant="body2">
                  <b>{mood}</b>
                </Typography>
                <Typography className={classes.muted} variant="caption">
                  Mood
                </Typography>
              </div>
            </div>
          ))}
        </Paper>
        <Paper className={`${classes.panel} ${classes.tracks}`}>
          <Box p={2}>
            <Typography className={classes.eyebrow}>All songs</Typography>
            <Typography className={classes.title} variant="h5">
              Library
            </Typography>
          </Box>
          <TrackRows songs={songs} />
        </Paper>
      </div>
    </div>
  )
}

const VariantC = ({ songs, current, genres, moods }) => {
  const classes = useStyles()
  return (
    <div className={classes.cRoot}>
      <Paper className={`${classes.panel} ${classes.cNav}`}>
        <Typography className={classes.eyebrow}>Browse</Typography>
        <Box mt={2}>
          <div className={`${classes.cNavItem} ${classes.cActive}`}>
            <LibraryMusicIcon fontSize="small" /> Songs
          </div>
          <div className={classes.cNavItem}>
            <AlbumIcon fontSize="small" /> Albums
          </div>
          <div className={classes.cNavItem}>
            <CategoryIcon fontSize="small" /> Categories
          </div>
          <div className={classes.cNavItem}>
            <WbSunnyOutlinedIcon fontSize="small" /> Moods
          </div>
          <div className={classes.cNavItem}>
            <QueueMusicIcon fontSize="small" /> Playlists
          </div>
        </Box>
        <Box mt={4}>
          <Typography className={classes.eyebrow}>Quick filters</Typography>
          <div className={classes.chips} style={{ marginTop: 12 }}>
            {[...genres, ...moods].slice(0, 5).map((item) => (
              <Chip
                size="small"
                className={classes.chip}
                key={item}
                label={item}
              />
            ))}
          </div>
        </Box>
      </Paper>
      <main>
        <div className={classes.cDock}>
          <Artwork
            record={current}
            size={120}
            square
            className={classes.cDockCover}
            title={current?.title}
          />
          <div style={{ minWidth: 0 }}>
            <Typography className={classes.trackTitle}>
              {current?.title || 'Nothing playing'}
            </Typography>
            <Typography className={classes.muted} variant="caption">
              {current?.artist}
            </Typography>
          </div>
          <LinearProgress
            className={`${classes.progress} ${classes.hideSmall}`}
            variant="determinate"
            value={36}
          />
          <div
            className={classes.controls}
            style={{ margin: 0, gap: 10 }}
            aria-hidden="true"
          >
            <SkipPreviousRoundedIcon fontSize="small" />
            <span className={classes.play} style={{ width: 38, height: 38 }}>
              <PauseRoundedIcon fontSize="small" />
            </span>
            <SkipNextRoundedIcon fontSize="small" />
          </div>
        </div>
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="flex-end"
          mb={2}
          style={{ gap: 16 }}
        >
          <div>
            <Typography className={classes.eyebrow}>
              Complete collection
            </Typography>
            <Typography className={classes.title} variant="h3">
              Songs
            </Typography>
          </div>
          <div className={classes.cSearch}>
            <SearchIcon fontSize="small" /> Search library
          </div>
        </Box>
        <Paper className={`${classes.panel} ${classes.tracks}`}>
          <TrackRows songs={songs} />
        </Paper>
      </main>
    </div>
  )
}

export const SongListPrototype = () => {
  const classes = useStyles()
  const location = useLocation()
  const requestedVariant = new URLSearchParams(location.search).get('variant')
  const variant = ['A', 'B', 'C'].includes(requestedVariant)
    ? requestedVariant
    : 'A'
  const { data, ids, loading } = useListContext()
  const playing = useSelector((state) => state.player.current?.song)
  const songs = useMemo(
    () =>
      (ids || [])
        .map((id) => data?.[id])
        .filter(Boolean)
        .slice(0, 12),
    [data, ids],
  )
  const current = playing || songs[0]
  const genres = useMemo(() => valuesFor(songs, 'genre'), [songs])
  const moods = useMemo(() => valuesFor(songs, 'mood'), [songs])
  const shared = { songs, current, genres, moods }

  if (loading && songs.length === 0) return <LinearProgress />

  return (
    <div className={classes.root}>
      {variant === 'A' && <VariantA {...shared} />}
      {variant === 'B' && <VariantB {...shared} />}
      {variant === 'C' && <VariantC {...shared} />}
      <PrototypeSwitcher names={variantNames} variant={variant} />
    </div>
  )
}
