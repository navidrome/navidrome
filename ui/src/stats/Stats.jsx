import React, { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, Typography, Box, Tabs, Tab, CircularProgress, Alert, Select, MenuItem, FormControl, InputLabel } from '@mui/material'
import { Timeline, MusicNote, Album, Person } from '@mui/icons-material'
import { fetchWithAuth } from '../utils/fetchWithAuth'

const TIME_RANGES = [
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: '90d', label: 'Last 90 days' },
  { value: '1y', label: 'Last year' },
  { value: 'all', label: 'All time' },
]

function TopItemCard({ item, rank, icon: Icon }) {
  return (
    <Card sx={{ mb: 1, display: 'flex', alignItems: 'center', gap: 2, p: 1 }}>
      <Typography variant="h5" sx={{ minWidth: 40, color: 'text.secondary', fontWeight: 'bold' }}>
        #{rank}
      </Typography>
      {item.imagePath && (
        <Box
          component="img"
          src={`${process.env.REACT_APP_BASE_URL || ''}/api/img/300/300/${item.imagePath}`}
          sx={{ width: 48, height: 48, borderRadius: 1, objectFit: 'cover' }}
        />
      )}
      {!item.imagePath && (
        <Box sx={{ width: 48, height: 48, borderRadius: 1, bgcolor: 'action.hover', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Icon sx={{ color: 'text.secondary' }} />
        </Box>
      )}
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography variant="body1" noWrap sx={{ fontWeight: 500 }}>{item.name}</Typography>
        <Typography variant="caption" color="text.secondary">
          {item.playCount?.toLocaleString()} plays · {(item.totalMsPlayed / 60000).toFixed(0)} min
        </Typography>
      </Box>
    </Card>
  )
}

export default function Stats() {
  const [tab, setTab] = useState(0)
  const [timeRange, setTimeRange] = useState('30d')
  const [artists, setArtists] = useState([])
  const [albums, setAlbums] = useState([])
  const [songs, setSongs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const loadStats = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [artistsRes, albumsRes, songsRes] = await Promise.all([
        fetchWithAuth(`/api/listening-stats/artists?timeRange=${timeRange}&limit=20`),
        fetchWithAuth(`/api/listening-stats/albums?timeRange=${timeRange}&limit=20`),
        fetchWithAuth(`/api/listening-stats/songs?timeRange=${timeRange}&limit=20`),
      ])
      if (!artistsRes.ok) throw new Error('Failed to load artist stats')
      const [a, al, s] = await Promise.all([artistsRes.json(), albumsRes.json(), songsRes.json()])
      setArtists(a || [])
      setAlbums(al || [])
      setSongs(s || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }, [timeRange])

  useEffect(() => { loadStats() }, [loadStats])

  return (
    <Box sx={{ p: 3, maxWidth: 800, mx: 'auto' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Timeline /> Listening Statistics
        </Typography>
        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>Time Range</InputLabel>
          <Select value={timeRange} label="Time Range" onChange={(e) => setTimeRange(e.target.value)}>
            {TIME_RANGES.map(r => <MenuItem key={r.value} value={r.value}>{r.label}</MenuItem>)}
          </Select>
        </FormControl>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Card sx={{ mb: 2 }}>
        <Tabs value={tab} onChange={(_, v) => setTab(v)} centered>
          <Tab icon={<Person />} label="Artists" />
          <Tab icon={<Album />} label="Albums" />
          <Tab icon={<MusicNote />} label="Songs" />
        </Tabs>
        <CardContent>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}><CircularProgress /></Box>
          ) : (
            <Box>
              {tab === 0 && artists.map((item, i) => <TopItemCard key={item.id} item={item} rank={i + 1} icon={Person} />)}
              {tab === 0 && artists.length === 0 && <Typography color="text.secondary" align="center" py={4}>No artist plays yet. Start listening!</Typography>}
              {tab === 1 && albums.map((item, i) => <TopItemCard key={item.id} item={item} rank={i + 1} icon={Album} />)}
              {tab === 1 && albums.length === 0 && <Typography color="text.secondary" align="center" py={4}>No album plays yet.</Typography>}
              {tab === 2 && songs.map((item, i) => <TopItemCard key={item.id} item={item} rank={i + 1} icon={MusicNote} />)}
              {tab === 2 && songs.length === 0 && <Typography color="text.secondary" align="center" py={4}>No song plays yet.</Typography>}
            </Box>
          )}
        </CardContent>
      </Card>
    </Box>
  )
}
