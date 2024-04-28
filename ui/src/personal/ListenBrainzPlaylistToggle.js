import { useEffect, useState } from 'react'
import { useNotify, useTranslate } from 'react-admin'
import { FormControl, FormControlLabel, Switch } from '@material-ui/core'
import { httpClient } from '../dataProvider'

export const ListenBrainzPlaylistToggle = () => {
  const notify = useNotify()
  const translate = useTranslate()
  const [linked, setLinked] = useState(null)

  const togglePlaylist = () => {
    if (linked) {
      httpClient('/api/listenbrainz/playlist', { method: 'DELETE' })
        .then(() => {
          setLinked(false)
          notify('message.agentRecommendedUnsyncSuccess', 'success', {
            agent: 'ListenBrainz',
          })
        })
        .catch((error) =>
          notify('message.agentRecommendedUnsyncFail', 'warning', {
            agent: 'ListenBrainz',
            error: error.body?.error || error.message,
          }),
        )
    } else {
      httpClient('/api/listenbrainz/playlist', { method: 'PUT' })
        .then(() => {
          setLinked(true)
          notify('message.agentRecommendedSyncSuccess', 'success', {
            agent: 'ListenBrainz',
          })
        })
        .catch((error) =>
          notify('message.agentRecommendedSyncFail', 'warning', {
            agent: 'ListenBrainz',
            error: error.body?.error || error.message,
          }),
        )
    }
  }

  useEffect(() => {
    httpClient('/api/listenbrainz/playlist')
      .then((response) => {
        setLinked(response.json.status === true)
      })
      .catch(() => {
        setLinked(false)
      })
  }, [])

  return (
    <>
      <FormControl>
        <FormControlLabel
          control={
            <Switch
              id={'listenbrainz-playlist'}
              color="primary"
              checked={linked === true}
              disabled={linked === null}
              onChange={togglePlaylist}
            />
          }
          label={
            <span>
              {translate('menu.personal.options.listenBrainzPlaylistSync')}
            </span>
          }
        />
      </FormControl>
    </>
  )
}
