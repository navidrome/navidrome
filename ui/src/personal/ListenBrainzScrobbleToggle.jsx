import { useEffect, useState } from 'react'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import { FormControl, FormControlLabel, Switch } from '@material-ui/core'
import { ListenBrainzTokenDialog } from '../dialogs'
import { useDispatch } from 'react-redux'
import { openListenBrainzTokenDialog } from '../actions'

export const ListenBrainzScrobbleToggle = () => {
  const dispatch = useDispatch()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const translate = useTranslate()
  const [linked, setLinked] = useState(null)

  const toggleScrobble = () => {
    if (linked) {
      dataProvider
        .unlinkListenBrainz()
        .then(() => {
          setLinked(false)
          notify('message.listenBrainzUnlinkSuccess', 'success')
        })
        .catch(() => notify('message.listenBrainzUnlinkFailure', 'warning'))
    } else {
      dispatch(openListenBrainzTokenDialog())
    }
  }

  useEffect(() => {
    dataProvider
      .getListenBrainzLink()
      .then((response) => {
        setLinked(response.data.status === true)
      })
      .catch(() => {
        setLinked(false)
      })
  }, [dataProvider])

  return (
    <>
      <FormControl>
        <FormControlLabel
          control={
            <Switch
              id={'listenbrainz'}
              color="primary"
              checked={linked === true}
              disabled={linked === null}
              onChange={toggleScrobble}
            />
          }
          label={
            <span>
              {translate('menu.personal.options.listenBrainzScrobbling')}
            </span>
          }
        />
      </FormControl>
      <ListenBrainzTokenDialog setLinked={setLinked} />
    </>
  )
}
