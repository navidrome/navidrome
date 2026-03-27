import { useEffect, useRef, useState } from 'react'
import { useNotify, useTranslate } from 'react-admin'
import {
  FormControl,
  FormControlLabel,
  FormHelperText,
  LinearProgress,
  Switch,
  Tooltip,
} from '@material-ui/core'
import { useInterval } from '../common'
import { baseUrl, openInNewTab } from '../utils'
import { httpClient } from '../dataProvider'

const buildAuthUrl = (authUrl, apiKey, callbackUrl) => {
  const url = new URL(authUrl)
  url.searchParams.set('api_key', apiKey)
  url.searchParams.set('cb', callbackUrl)
  return url.toString()
}

const Progress = (props) => {
  const { setLinked, setCheckingLink, apiKey, authUrl } = props
  const notify = useNotify()
  let linkCheckDelay = 2000
  let linkChecks = 30
  const openedTab = useRef()

  useEffect(() => {
    const callbackEndpoint = baseUrl(
      `/api/lastfm/link/callback?uid=${localStorage.getItem('userId')}`,
    )
    const callbackUrl = `${window.location.origin}${callbackEndpoint}`
    try {
      openedTab.current = openInNewTab(
        buildAuthUrl(authUrl, apiKey, callbackUrl),
      )
    } catch {
      setCheckingLink(false)
      setLinked(false)
      notify('message.lastfmLinkFailure', 'warning')
    }
  }, [apiKey, authUrl])

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
  const [apiKey, setApiKey] = useState(false)
  const [authUrl, setAuthUrl] = useState(null)

  useEffect(() => {
    httpClient('/api/lastfm/link')
      .then((response) => {
        setLinked(response.json.status === true)
        setApiKey(response.json.apiKey)
        setAuthUrl(response.json.authUrl || null)
      })
      .catch(() => {
        setLinked(false)
      })
  }, [setLinked, setApiKey])

  const toggleScrobble = () => {
    if (!linked) {
      if (!authUrl) {
        notify('message.lastfmLinkFailure', 'warning')
        return
      }
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

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={'lastfm'}
            color="primary"
            checked={linked || checkingLink}
            disabled={!apiKey || !authUrl || linked === null || checkingLink}
            onChange={toggleScrobble}
          />
        }
        label={
          <span>{translate('menu.personal.options.lastfmScrobbling')}</span>
        }
      />
      {checkingLink && (
        <Progress
          setLinked={setLinked}
          setCheckingLink={setCheckingLink}
          apiKey={apiKey}
          authUrl={authUrl}
        />
      )}
      {!apiKey && (
        <FormHelperText id="scrobble-lastfm-disabled-helper-text">
          {translate('menu.personal.options.lastfmNotConfigured')}
        </FormHelperText>
      )}
    </FormControl>
  )
}
