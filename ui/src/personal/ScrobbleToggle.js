import { useState } from 'react'
import { useNotify, useTranslate } from 'react-admin'
import {
  FormControl,
  FormControlLabel,
  LinearProgress,
  Switch,
} from '@material-ui/core'
import { useInterval } from '../common'
import { baseUrl } from '../utils'

const Progress = (props) => {
  const { setLinked, setCheckingLink } = props
  const translate = useTranslate()
  const notify = useNotify()
  let linkCheckDelay = 2000
  let linkChecks = 5
  // openInNewTab(
  //   '/api/lastfm/link'
  // )

  const endChecking = (success) => {
    linkCheckDelay = null
    setCheckingLink(false)
    if (success) {
      notify(translate('Last.fm successfully linked!'), 'success')
    } else {
      notify(translate('Last.fm could not be linked'), 'warning')
    }
    setLinked(success)
  }

  useInterval(() => {
    fetch(baseUrl(`/api/lastfm/link/status?c=${linkChecks}`))
      .then((response) => response.text())
      .then((response) => {
        if (response === 'true') {
          endChecking(true)
        }
      })
      .catch((error) => {
        endChecking(false)
        throw new Error(error)
      })

    if (--linkChecks === 0) {
      endChecking(false)
    }
  }, linkCheckDelay)

  return linkChecks > 0 && <LinearProgress />
}

export const ScrobbleToggle = (props) => {
  const translate = useTranslate()
  const [linked, setLinked] = useState(false)
  const [checkingLink, setCheckingLink] = useState(false)
  const toggleScrobble = () => {
    if (!linked) {
      return setCheckingLink(true)
    }
    setLinked(!linked)
  }

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={'notifications'}
            color="primary"
            checked={linked || checkingLink}
            disabled={checkingLink}
            onChange={toggleScrobble}
          />
        }
        label={<span>{translate('Scrobble to Last.FM')}</span>}
      />
      {checkingLink && (
        <Progress setLinked={setLinked} setCheckingLink={setCheckingLink} />
      )}
    </FormControl>
  )
}
