import { useNotify, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { setNotificationsState } from '../actions'
import {
  FormControl,
  FormControlLabel,
  LinearProgress,
  Switch,
} from '@material-ui/core'
import { useState } from 'react'
import { openInNewTab } from '../utils'

export const ScrobbleToggle = (props) => {
  const translate = useTranslate()
  const [linked, setLinked] = useState(false)

  const toggleScrobble = (event) => {
    if (!linked) {
      openInNewTab(
        'https://www.last.fm/api/auth/?api_key=c2918986bf01b6ba353c0bc1bdd27bea'
      )
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
            checked={linked}
            disabled={linked}
            onChange={toggleScrobble}
          />
        }
        label={<span>{translate('Scrobble to Last.FM')}</span>}
      />
      {linked && <LinearProgress />}
    </FormControl>
  )
}
