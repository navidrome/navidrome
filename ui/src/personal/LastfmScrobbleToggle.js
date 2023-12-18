import { useEffect, useRef, useState } from 'react'
import { useNotify, useTranslate } from 'react-admin'
import {
  FormControl,
  FormControlLabel,
  LinearProgress,
  Switch,
} from '@material-ui/core'
import { useInterval } from '../common'
import config from '../config'
import { baseUrl, openInNewTab } from '../utils'
import { httpClient } from '../dataProvider'

const Progress = (props) => {
  const { setLinked, setCheckingLink } = props
  const notify = useNotify()
  let linkCheckDelay = 2000
  let linkChecks = 30
  const openedTab = useRef()

  useEffect(() => {
    const callbackEndpoint = baseUrl(
      `/api/lastfm/link/callback?uid=${localStorage.getItem('userId')}`,
    )
    const callbackUrl = `${window.location.origin}${callbackEndpoint}`
    openedTab.current = openInNewTab(
      `https://www.last.fm/api/auth/?api_key=${config.lastFMApiKey}&cb=${callbackUrl}`,
    )
  }, [])

  const endChecking = (success) => {
    linkCheckDelay = null
    setCheckingLink(false)
    if (success) {
      notify('message.lastfmLinkSuccess', 'success')
    } else {
      notify('message.lastfmLinkFailure', 'warning')
    }
    setLinked(success)
  }

  useInterval(() => {
    httpClient('/api/lastfm/link')
      .then((response) => {
        let result = false
        if (response.json.status === true) {
          result = true
          endChecking(true)
        }
        return result
      })
      .then((result) => {
        if (!result && openedTab.current?.closed === true) {
          endChecking(false)
          result = true
        }
        return result
      })
      .then((result) => {
        if (!result && --linkChecks === 0) {
          endChecking(false)
        }
      })
      .catch(() => {
        endChecking(false)
      })
  }, linkCheckDelay)

  return <LinearProgress />
}

export const LastfmScrobbleToggle = (props) => {
  const notify = useNotify()
  const translate = useTranslate()
  const [linked, setLinked] = useState(null)
  const [checkingLink, setCheckingLink] = useState(false)

  const toggleScrobble = () => {
    if (!linked) {
      setCheckingLink(true)
    } else {
      httpClient('/api/lastfm/link', { method: 'DELETE' })
        .then(() => {
          setLinked(false)
          notify('message.lastfmUnlinkSuccess', 'success')
        })
        .catch(() => notify('message.lastfmUnlinkFailure', 'warning'))
    }
  }

  useEffect(() => {
    httpClient('/api/lastfm/link')
      .then((response) => {
        setLinked(response.json.status === true)
      })
      .catch(() => {
        setLinked(false)
      })
  }, [])

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={'lastfm'}
            color="primary"
            checked={linked || checkingLink}
            disabled={linked === null || checkingLink}
            onChange={toggleScrobble}
          />
        }
        label={
          <span>{translate('menu.personal.options.lastfmScrobbling')}</span>
        }
      />
      {checkingLink && (
        <Progress setLinked={setLinked} setCheckingLink={setCheckingLink} />
      )}
    </FormControl>
  )
}
