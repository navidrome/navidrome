import React, { useEffect, useState } from 'react'
import {
  Avatar,
  Card,
  CardContent,
  CircularProgress,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Typography,
  makeStyles,
} from '@material-ui/core'
import Rating from '@material-ui/lab/Rating'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import ArrowBackIcon from '@material-ui/icons/ArrowBack'
import { Link } from 'react-router-dom'
import { Title, useTranslate } from 'react-admin'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'
import subsonic from '../subsonic'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  root: { padding: theme.spacing(2) },
  back: {
    display: 'flex',
    alignItems: 'center',
    color: theme.palette.primary.main,
    textDecoration: 'none',
    marginBottom: theme.spacing(2),
    '&:hover': { textDecoration: 'underline' },
  },
  backIcon: { marginRight: theme.spacing(0.5), fontSize: 18 },
  header: { marginBottom: theme.spacing(2) },
  avatar: { width: 48, height: 48, borderRadius: 4 },
  listItem: { paddingLeft: 0, paddingRight: 0 },
}))

const UserRatingItems = ({ match }) => {
  const { userId, userName, type, rating } = match.params
  const classes = useStyles()
  const translate = useTranslate()
  const [items, setItems] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    httpClient(
      `${REST_URL}/ratingItems?userId=${encodeURIComponent(userId)}&type=${type}&rating=${rating}`,
    )
      .then(({ json }) => setItems(Array.isArray(json) ? json : []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [userId, type, rating])

  const typeLabel =
    type === 'album'
      ? translate('resources.album.name', { smart_count: 2 })
      : translate('resources.song.name', { smart_count: 2 })

  const title = `${userName} · ${typeLabel} · ${rating}★`

  const getCoverUrl = (item) => {
    const prefix = type === 'album' ? 'al-' : 'mf-'
    return subsonic.getCoverArtUrl(
      { id: item.id, albumArtist: type === 'album' ? item.artist : undefined, album: type === 'song' ? item.name : undefined, updatedAt: item.updatedAt },
      40,
    )
  }

  return (
    <div className={classes.root}>
      <Title title={title} />
      <Link to="/userRatings" className={classes.back}>
        <ArrowBackIcon className={classes.backIcon} />
        {translate('menu.userRatings')}
      </Link>
      <div className={classes.header}>
        <Typography variant="h5" gutterBottom>
          {userName}
        </Typography>
        <Typography variant="subtitle1" color="textSecondary">
          {typeLabel} ·{' '}
          <Rating
            value={parseInt(rating)}
            readOnly
            size="small"
            style={{ verticalAlign: 'middle' }}
            icon={<StarIcon fontSize="inherit" style={{ color: '#ffb400' }} />}
            emptyIcon={<StarBorderIcon fontSize="inherit" style={{ color: '#ffb400', opacity: 0.4 }} />}
          />
        </Typography>
      </div>
      <Card>
        <CardContent>
          {loading && <CircularProgress />}
          {error && <Typography color="error">{error}</Typography>}
          {items && items.length === 0 && (
            <Typography color="textSecondary">No items found.</Typography>
          )}
          {items && items.length > 0 && (
            <List disablePadding>
              {items.map((item) => {
                const albumId = type === 'album' ? item.id : item.albumId
                return (
                  <ListItem
                    key={item.id}
                    className={classes.listItem}
                    divider
                    button
                    component={Link}
                    to={`/album/${albumId}/show`}
                  >
                    <ListItemAvatar>
                      <Avatar
                        src={getCoverUrl(item)}
                        variant="square"
                        className={classes.avatar}
                      />
                    </ListItemAvatar>
                    <ListItemText
                      primary={item.name}
                      secondary={item.artist}
                    />
                  </ListItem>
                )
              })}
            </List>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default UserRatingItems
