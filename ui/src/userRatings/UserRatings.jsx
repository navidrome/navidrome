import React, { useEffect, useState } from 'react'
import {
  Card,
  CardContent,
  CircularProgress,
  Divider,
  Tab,
  Tabs,
  Typography,
  makeStyles,
} from '@material-ui/core'
import Rating from '@material-ui/lab/Rating'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import PeopleIcon from '@material-ui/icons/People'
import { useHistory } from 'react-router-dom'
import { Title, useTranslate } from 'react-admin'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(2),
  },
  userCard: {
    marginBottom: theme.spacing(3),
  },
  userName: {
    marginBottom: theme.spacing(1),
  },
  totalLabel: {
    color: theme.palette.primary.main,
    fontWeight: 'bold',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
  },
  row: {
    '&:nth-child(even)': {
      backgroundColor: theme.palette.action.hover,
    },
  },
  ratingCell: {
    width: 60,
    textAlign: 'right',
    paddingRight: theme.spacing(1),
    color: theme.palette.primary.main,
    fontWeight: 'bold',
    whiteSpace: 'nowrap',
  },
  countCell: {
    width: 50,
    textAlign: 'right',
    paddingRight: theme.spacing(1),
    fontWeight: 'bold',
    whiteSpace: 'nowrap',
  },
  barCell: {
    padding: `${theme.spacing(0.5)}px ${theme.spacing(1)}px`,
  },
  barOuter: {
    height: 20,
    borderRadius: 3,
    overflow: 'hidden',
    backgroundColor: theme.palette.action.selected,
    minWidth: 4,
  },
  barInner: {
    height: '100%',
    borderRadius: 3,
    minWidth: 4,
    transition: 'width 0.4s ease',
  },
  starsCell: {
    width: 110,
    paddingLeft: theme.spacing(1),
  },
  emptyMsg: {
    color: theme.palette.text.secondary,
    fontStyle: 'italic',
    marginTop: theme.spacing(1),
  },
}))

const ratingColor = (rating) => {
  if (rating >= 4) return '#69c76f'
  if (rating >= 3) return '#b8c750'
  if (rating >= 2) return '#d4a843'
  return '#c75050'
}

const ALL_RATINGS = [5, 4, 3, 2, 1]

const RatingTable = ({ stats, label, userId, userName, type }) => {
  const classes = useStyles()
  const history = useHistory()

  const countMap = {}
  let total = 0
  ;(stats || []).forEach(({ rating, count }) => {
    countMap[rating] = count
    total += count
  })

  const maxCount = Math.max(...Object.values(countMap), 1)

  return (
    <div>
      <Typography variant="subtitle1">
        {label}:{' '}
        <span className={classes.totalLabel}>{total}</span>
      </Typography>
      {total === 0 ? (
        <Typography className={classes.emptyMsg}>No ratings yet</Typography>
      ) : (
        <table className={classes.table}>
          <tbody>
            {ALL_RATINGS.map((r) => {
              const count = countMap[r] || 0
              const width = count > 0 ? Math.max((count / maxCount) * 100, 2) : 2
              const linkTo = count > 0
                ? `/userRatings/${encodeURIComponent(userId)}/${encodeURIComponent(userName)}/${type}/${r}`
                : null
              return (
                <tr
                  key={r}
                  className={classes.row}
                  style={{ cursor: count > 0 ? 'pointer' : 'default' }}
                  onClick={() => count > 0 && history.push(linkTo)}
                >
                  <td className={classes.ratingCell}>{r}.0</td>
                  <td className={classes.countCell}>{count}</td>
                  <td className={classes.barCell}>
                    <div className={classes.barOuter}>
                      <div
                        className={classes.barInner}
                        style={{
                          width: `${width}%`,
                          backgroundColor: count > 0 ? ratingColor(r) : 'transparent',
                          opacity: count > 0 ? 1 : 0.2,
                        }}
                      />
                    </div>
                  </td>
                  <td className={classes.starsCell}>
                    <Rating
                      value={r}
                      readOnly
                      size="small"
                      icon={<StarIcon fontSize="inherit" style={{ color: '#ffb400' }} />}
                      emptyIcon={<StarBorderIcon fontSize="inherit" style={{ color: '#ffb400', opacity: 0.4 }} />}
                    />
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </div>
  )
}

const UserRatingCard = ({ user }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [tab, setTab] = useState(0)

  return (
    <Card className={classes.userCard}>
      <CardContent>
        <Typography variant="h6" className={classes.userName}>
          <PeopleIcon fontSize="small" style={{ verticalAlign: 'middle', marginRight: 6 }} />
          {user.userName}
        </Typography>
        <Tabs
          value={tab}
          onChange={(_, v) => setTab(v)}
          indicatorColor="primary"
          textColor="primary"
          variant="scrollable"
        >
          <Tab label={translate('resources.song.name', { smart_count: 2 })} />
          <Tab label={translate('resources.album.name', { smart_count: 2 })} />
        </Tabs>
        <Divider />
        <div style={{ marginTop: 12 }}>
          {tab === 0 && (
            <RatingTable
              stats={user.songStats}
              label={translate('resources.song.name', { smart_count: 2 })}
              userId={user.userId}
              userName={user.userName}
              type="song"
            />
          )}
          {tab === 1 && (
            <RatingTable
              stats={user.albumStats}
              label={translate('resources.album.name', { smart_count: 2 })}
              userId={user.userId}
              userName={user.userName}
              type="album"
            />
          )}
        </div>
      </CardContent>
    </Card>
  )
}

const UserRatings = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    httpClient(`${REST_URL}/ratingStats`)
      .then(({ json }) => {
        setData(Array.isArray(json) ? json : [])
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className={classes.root}>
      <Title title={translate('menu.userRatings')} />
      <Typography variant="h5" gutterBottom>
        {translate('menu.userRatings')}
      </Typography>
      {loading && <CircularProgress />}
      {error && <Typography color="error">{error}</Typography>}
      {data && data.length === 0 && (
        <Typography color="textSecondary">No ratings yet.</Typography>
      )}
      {data &&
        data.map((user) => <UserRatingCard key={user.userId} user={user} />)}
    </div>
  )
}

export default UserRatings
