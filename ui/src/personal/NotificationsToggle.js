import { useNotify, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { setNotificationsState } from '../actions'
import {
  FormControl,
  FormControlLabel,
  FormHelperText,
  Switch,
} from '@material-ui/core'

export const NotificationsToggle = () => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const notify = useNotify()
  const currentSetting = useSelector((state) => state.settings.notifications)
  const notAvailable = !('Notification' in window) || !window.isSecureContext

  if (
    (currentSetting && Notification.permission !== 'granted') ||
    notAvailable
  ) {
    dispatch(setNotificationsState(false))
  }

  const toggleNotifications = (event) => {
    if (currentSetting && !event.target.checked) {
      dispatch(setNotificationsState(false))
    } else {
      if (Notification.permission === 'denied') {
        notify(translate('message.notifications_blocked'), 'warning')
      } else {
        Notification.requestPermission().then((permission) => {
          dispatch(setNotificationsState(permission === 'granted'))
        })
      }
    }
  }

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={'notifications'}
            color="primary"
            checked={currentSetting}
            disabled={notAvailable}
            onChange={toggleNotifications}
          />
        }
        label={
          <span>
            {translate('menu.personal.options.desktop_notifications')}
          </span>
        }
      />
      {notAvailable && (
        <FormHelperText id="notifications-disabled-helper-text">
          {translate('message.notifications_not_available')}
        </FormHelperText>
      )}
    </FormControl>
  )
}
