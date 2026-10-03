/**
 * Journey history: every generated journey with its settings and outcome,
 * plus Replay / Regenerate / Delete / Like actions.
 */
import {
  Card,
  CardContent,
  Chip,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Typography,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import RefreshIcon from '@material-ui/icons/Refresh'
import DeleteIcon from '@material-ui/icons/Delete'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'

const useStyles = makeStyles((theme) => ({
  card: { marginTop: theme.spacing(3) },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  title: { fontWeight: 700 },
  list: { padding: 0 },
  item: {
    alignItems: 'flex-start',
    paddingRight: theme.spacing(1),
  },
  meta: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(0.5),
    marginTop: theme.spacing(0.5),
  },
  chip: { height: 20, fontSize: 11 },
  status: {
    fontSize: 11,
    color: theme.palette.text.hint,
    textTransform: 'capitalize',
  },
  actions: { display: 'flex', alignItems: 'center' },
  empty: {
    color: theme.palette.text.secondary,
    padding: theme.spacing(2),
    textAlign: 'center',
  },
}))

const STATUS_LABELS = {
  active: 'In progress',
  completed: 'Completed',
  interrupted: 'Interrupted',
  stopped: 'Stopped early',
}

/**
 * @param {Object} props
 * @param {Array} props.entries history entries (newest first)
 * @param {Function} props.onReplay
 * @param {Function} props.onRegenerate loads the entry's settings into the creator
 * @param {Function} props.onDelete
 * @param {Function} props.onToggleLike
 */
export const JourneyHistory = ({
  entries,
  onReplay,
  onRegenerate,
  onDelete,
  onToggleLike,
}) => {
  const classes = useStyles()

  return (
    <Card className={classes.card} variant="outlined">
      <CardContent>
        <div className={classes.header}>
          <Typography variant="h6" className={classes.title}>
            Journey History
          </Typography>
        </div>
        {!entries.length ? (
          <div className={classes.empty}>
            Journeys you create will appear here, ready to replay or refine.
          </div>
        ) : (
          <List className={classes.list} aria-label="Journey history">
            {entries.map((entry) => (
              <ListItem key={entry.id} className={classes.item} divider>
                <IconButton
                  onClick={() => onReplay(entry)}
                  aria-label={`Replay journey ${entry.title}`}
                  size="small"
                >
                  <PlayArrowIcon />
                </IconButton>
                <ListItemText
                  primary={entry.title}
                  secondary={
                    <>
                      {new Date(entry.createdAt).toLocaleString()}
                      <div className={classes.meta}>
                        <Chip
                          className={classes.chip}
                          size="small"
                          label={entry.mood}
                        />
                        <Chip
                          className={classes.chip}
                          size="small"
                          variant="outlined"
                          label={`${entry.durationMinutes} min`}
                        />
                        <Chip
                          className={classes.chip}
                          size="small"
                          variant="outlined"
                          label={`${entry.trackCount} tracks`}
                        />
                        {entry.genre && (
                          <Chip
                            className={classes.chip}
                            size="small"
                            variant="outlined"
                            label={entry.genre}
                          />
                        )}
                        {entry.skipped > 0 && (
                          <Chip
                            className={classes.chip}
                            size="small"
                            variant="outlined"
                            label={`${entry.skipped} skipped`}
                          />
                        )}
                        <span className={classes.status}>
                          {STATUS_LABELS[entry.status] || entry.status}
                        </span>
                      </div>
                    </>
                  }
                />
                <div className={classes.actions}>
                  <IconButton
                    onClick={() => onToggleLike(entry.id)}
                    aria-label={
                      entry.liked
                        ? 'Remove from favourites'
                        : 'Mark as favourite'
                    }
                    size="small"
                  >
                    {entry.liked ? (
                      <FavoriteIcon color="secondary" fontSize="small" />
                    ) : (
                      <FavoriteBorderIcon fontSize="small" />
                    )}
                  </IconButton>
                  <IconButton
                    onClick={() => onRegenerate(entry)}
                    aria-label={`Regenerate a journey like ${entry.title}`}
                    size="small"
                  >
                    <RefreshIcon fontSize="small" />
                  </IconButton>
                  <IconButton
                    onClick={() => onDelete(entry.id)}
                    aria-label={`Delete journey ${entry.title}`}
                    size="small"
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </div>
              </ListItem>
            ))}
          </List>
        )}
      </CardContent>
    </Card>
  )
}

export default JourneyHistory
