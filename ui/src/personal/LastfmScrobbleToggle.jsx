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

const Progress = (props) => {
  const { setLinked, setCheckingLink, openedTab } = props
  const notify = useNotify()
  let linkCheckDelay = 2000
  let linkChecks = 30

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
  const openedTab = useRef()

  useEffect(() => {
    httpClient('/api/lastfm/link')
      .then((response) => {
        setLinked(response.json.status === true)
        setApiKey(response.json.apiKey)
      })
      .catch(() => {
        setLinked(false)
      })
  }, [setLinked, setApiKey])

  const startLink = () => {
    // Open the tab synchronously so popup blockers attribute it to the click.
    let tab
    try {
      tab = openInNewTab('about:blank')
    } catch {
      notify('message.lastfmLinkFailure', 'warning')
      return
    }
    openedTab.current = tab
    setCheckingLink(true)
    httpClient('/api/lastfm/link')
      .then((response) => {
        const linkToken = response.json.linkToken
        if (!linkToken) {
          tab?.close()
          notify('message.lastfmLinkFailure', 'warning')
          setCheckingLink(false)
          return
        }
        const callbackEndpoint = baseUrl(
          `/api/lastfm/link/callback?uid=${encodeURIComponent(linkToken)}`,
        )
        const callbackUrl = `${window.location.origin}${callbackEndpoint}`
        tab.location.href = `https://www.last.fm/api/auth/?api_key=${apiKey}&cb=${callbackUrl}`
      })
      .catch(() => {
        tab?.close()
        notify('message.lastfmLinkFailure', 'warning')
        setCheckingLink(false)
      })
  }

  const toggleScrobble = () => {
    if (!linked) {
      startLink()
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
            disabled={!apiKey || linked === null || checkingLink}
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
          openedTab={openedTab}
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
