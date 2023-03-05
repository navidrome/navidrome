import { FormControl, FormControlLabel, Switch } from '@material-ui/core'
import { useEffect } from 'react'
import { useNotify, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { openWebhookDialog } from '../actions'
import { httpClient } from '../dataProvider'

export const WebhookToggle = ({ linked, name, url, setLinked }) => {
  const dispatch = useDispatch()
  const notify = useNotify()
  const translate = useTranslate()

  const toggleScrobble = () => {
    if (linked) {
      httpClient(`/api/webhook/${name}/link`, { method: 'DELETE' })
        .then(() => {
          setLinked(name, false)
          notify('message.webhookUnlinkSuccess', 'success', { name })
        })
        .catch(() =>
          notify('message.listenBrainzUnlinkFailure', 'warning', { name })
        )
    } else {
      dispatch(openWebhookDialog(name, url))
    }
  }

  useEffect(() => {
    httpClient(`/api/webhook/${name}/link`)
      .then((response) => {
        setLinked(name, response.json.status === true)
      })
      .catch(() => {
        setLinked(name, false)
      })
  }, [name, setLinked])

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={`webhook-${name}`}
            color="primary"
            checked={linked === true}
            disabled={linked === null}
            onChange={toggleScrobble}
          />
        }
        label={
          <span>
            {translate('menu.personal.options.webhookScrobbling', { name })}
          </span>
        }
      />
    </FormControl>
  )
}
