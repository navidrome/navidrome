import React, { useCallback, useState } from 'react'
import {
  Box,
  Button,
  CircularProgress,
  List,
  ListItem,
  ListItemText,
  TextField,
  Typography,
} from '@material-ui/core'
import { FormSpy } from 'react-final-form'
import { useNotify, useTranslate } from 'react-admin'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'

const listStyle = { maxHeight: 280, overflow: 'auto', marginTop: 8 }

const RadioBrowserSearchFields = ({ form }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [results, setResults] = useState([])
  const [searched, setSearched] = useState(false)

  const runSearch = useCallback(async () => {
    const q = query.trim()
    if (q.length < 2) {
      return
    }
    setLoading(true)
    setSearched(true)
    try {
      const { json } = await httpClient(
        `${REST_URL}/radio/browser/search?q=${encodeURIComponent(q)}`,
      )
      setResults(Array.isArray(json.stations) ? json.stations : [])
    } catch (_e) {
      setResults([])
      notify('resources.radio.radioBrowser.error', 'warning')
    } finally {
      setLoading(false)
    }
  }, [query, notify])

  const pickStation = useCallback(
    async (station) => {
      try {
        await httpClient(`${REST_URL}/radio/browser/click`, {
          method: 'POST',
          body: JSON.stringify({ streamUrl: station.streamUrl }),
          headers: new Headers({ 'Content-Type': 'application/json' }),
        })
      } catch (_e) {
        // best-effort popularity ping for radio-browser.info
      }
      form.change('name', station.name)
      form.change('streamUrl', station.streamUrl)
      form.change('homePageUrl', station.homePageUrl || '')
    },
    [form],
  )

  return (
    <Box marginBottom={2}>
      <Typography variant="subtitle2" color="textSecondary" gutterBottom>
        {translate('resources.radio.radioBrowser.hint')}
      </Typography>
      <Box display="flex" alignItems="flex-start" style={{ gap: 8 }}>
        <TextField
          label={translate('resources.radio.radioBrowser.placeholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              runSearch()
            }
          }}
          variant="outlined"
          size="small"
          fullWidth
          disabled={loading}
        />
        <Button
          variant="outlined"
          color="primary"
          onClick={runSearch}
          disabled={loading}
        >
          {loading ? (
            <CircularProgress size={20} />
          ) : (
            translate('resources.radio.radioBrowser.search')
          )}
        </Button>
      </Box>
      {searched && !loading && results.length === 0 && (
        <Typography
          variant="body2"
          color="textSecondary"
          style={{ marginTop: 8 }}
        >
          {translate('resources.radio.radioBrowser.noResults')}
        </Typography>
      )}
      {results.length > 0 && (
        <List dense disablePadding style={listStyle}>
          {results.map((s) => (
            <ListItem
              key={s.stationuuid || `${s.name}-${s.streamUrl}`}
              button
              onClick={() => pickStation(s)}
            >
              <ListItemText primary={s.name} secondary={s.streamUrl} />
            </ListItem>
          ))}
        </List>
      )}
    </Box>
  )
}

const RadioBrowserSearch = () => (
  <FormSpy subscription={{}}>
    {({ form }) => <RadioBrowserSearchFields form={form} />}
  </FormSpy>
)

export default RadioBrowserSearch
