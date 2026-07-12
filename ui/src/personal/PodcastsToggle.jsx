import { useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { setShowPodcasts } from '../actions'
import { FormControl, FormControlLabel, Switch } from '@material-ui/core'

export const PodcastsToggle = () => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const showPodcasts = useSelector((state) => state.settings.showPodcasts)

  const togglePodcasts = (event) => {
    dispatch(setShowPodcasts(event.target.checked))
  }

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={'showPodcasts'}
            color="primary"
            checked={showPodcasts !== false}
            onChange={togglePodcasts}
          />
        }
        label={<span>{translate('menu.personal.options.showPodcasts')}</span>}
      />
    </FormControl>
  )
}
