import React, { useState, useEffect } from 'react'
import { useTranslate, useDataProvider, Title } from 'react-admin'
import {
  Card,
  CardContent,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Avatar,
  Typography,
  CircularProgress,
  Box,
  Chip,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import MusicNoteIcon from '@material-ui/icons/MusicNote'
import AlbumIcon from '@material-ui/icons/Album'

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(2),
    maxWidth: 800,
    margin: '0 auto',
  },
  header: {
    marginBottom: theme.spacing(2),
  },
  listItem: {
    borderBottom: `1px solid ${theme.palette.divider}`,
    '&:last-child': {
      borderBottom: 'none',
    },
  },
  score: {
    marginLeft: theme.spacing(1),
  },
  avatar: {
    backgroundColor: theme.palette.primary.main,
  },
  emptyState: {
    textAlign: 'center',
    padding: theme.spacing(4),
    color: theme.palette.text.secondary,
  },
  modelInfo: {
    marginTop: theme.spacing(2),
    color: theme.palette.text.secondary,
    fontSize: '0.75rem',
  },
}))

const RecommendationList = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const [recommendations, setRecommendations] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [modelVersion, setModelVersion] = useState('')
  const [generatedAt, setGeneratedAt] = useState('')

  useEffect(() => {
    const fetchRecommendations = async () => {
      try {
        setLoading(true)
        const response = await fetch('/api/recommendation', {
          headers: {
            'x-nd-authorization': `Bearer ${localStorage.getItem('token')}`,
          },
        })
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`)
        }
        const data = await response.json()
        setRecommendations(data.recommendations || [])
        setModelVersion(data.modelVersion || '')
        setGeneratedAt(data.generatedAt || '')
      } catch (err) {
        console.error('Failed to fetch recommendations:', err)
        setError(err.message)
        setRecommendations([])
      } finally {
        setLoading(false)
      }
    }

    fetchRecommendations()
  }, [])

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
        <CircularProgress />
      </Box>
    )
  }

  return (
    <>
      <Title title="Recommendations" />
      <Card className={classes.root}>
        <CardContent>
          <Typography variant="h5" className={classes.header}>
            Recommended For You
          </Typography>

          {error && (
            <Typography color="error" gutterBottom>
              Could not load recommendations. The recommendation service may be
              starting up.
            </Typography>
          )}

          {!error && recommendations.length === 0 && (
            <div className={classes.emptyState}>
              <MusicNoteIcon style={{ fontSize: 48, opacity: 0.5 }} />
              <Typography variant="body1">
                No recommendations yet. Listen to some music and check back
                later!
              </Typography>
            </div>
          )}

          {recommendations.length > 0 && (
            <List>
              {recommendations.map((rec, index) => (
                <ListItem key={rec.id || index} className={classes.listItem}>
                  <ListItemAvatar>
                    <Avatar className={classes.avatar}>
                      <AlbumIcon />
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText
                    primary={rec.title}
                    secondary={`${rec.artist}${rec.album ? ` — ${rec.album}` : ''}`}
                  />
                  {rec.score && (
                    <Chip
                      label={`${Math.round(rec.score * 100)}%`}
                      size="small"
                      color="primary"
                      variant="outlined"
                      className={classes.score}
                    />
                  )}
                </ListItem>
              ))}
            </List>
          )}

          {modelVersion && (
            <Typography className={classes.modelInfo}>
              Model: {modelVersion}
              {generatedAt && ` · Generated: ${new Date(generatedAt).toLocaleString()}`}
            </Typography>
          )}
        </CardContent>
      </Card>
    </>
  )
}

export default RecommendationList
