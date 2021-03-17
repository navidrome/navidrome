import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  Card,
  FormControl,
  FormHelperText,
  FormControlLabel,
  Switch,
} from '@material-ui/core'
import {
  SelectInput,
  SimpleForm,
  Title,
  useLocale,
  useNotify,
  useSetLocale,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import HelpOutlineIcon from '@material-ui/icons/HelpOutline'
import {
  changeTheme,
  setMobileResolution,
  setNotificationsState,
} from '../actions'
import themes from '../themes'
import { docsUrl } from '../utils'
import { useGetLanguageChoices } from '../i18n'
import resolution from '../audioplayer/resolution'
import albumLists, { defaultAlbumList } from '../album/albumLists'

const useStyles = makeStyles({
  root: { marginTop: '1em' },
})

const helpKey = '_help'

function openInNewTab(url) {
  const win = window.open(url, '_blank')
  win.focus()
}

const HelpMsg = ({ caption }) => (
  <>
    <HelpOutlineIcon />
    &nbsp;&nbsp; {caption}
  </>
)

const SelectLanguage = (props) => {
  const translate = useTranslate()
  const setLocale = useSetLocale()
  const locale = useLocale()
  const { choices } = useGetLanguageChoices()

  choices.push({
    id: helpKey,
    name: <HelpMsg caption={'Help to translate'} />,
  })

  return (
    <SelectInput
      {...props}
      source="language"
      label={translate('menu.personal.options.language')}
      defaultValue={locale}
      choices={choices}
      translateChoice={false}
      onChange={(event) => {
        if (event.target.value === helpKey) {
          openInNewTab(docsUrl('/docs/developers/translations/'))
          return
        }
        setLocale(event.target.value)
        localStorage.setItem('locale', event.target.value)
      }}
    />
  )
}

const SelectTheme = (props) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const currentTheme = useSelector((state) => state.theme)
  const themeChoices = Object.keys(themes).map((key) => {
    return { id: key, name: themes[key].themeName }
  })
  themeChoices.push({
    id: helpKey,
    name: <HelpMsg caption={'Create your own'} />,
  })
  return (
    <SelectInput
      {...props}
      source="theme"
      label={translate('menu.personal.options.theme')}
      defaultValue={currentTheme}
      translateChoice={false}
      choices={themeChoices}
      onChange={(event) => {
        if (event.target.value === helpKey) {
          openInNewTab(docsUrl('/docs/developers/creating-themes/'))
          return
        }
        dispatch(changeTheme(event.target.value))
      }}
    />
  )
}

const SelectMobilePlayerResolution = (props) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  let currentResolution =
    useSelector((state) => state.settings.resolution) ||
    dispatch(setMobileResolution('MobileResolution'))

  currentResolution = useSelector((state) => state.settings.resolution)
  const resChoices = Object.keys(resolution).map((key) => {
    return { id: key, name: translate(`player.resolution.${key}`) }
  })

  return (
    <SelectInput
      {...props}
      label={translate('menu.personal.options.select_resolution')}
      defaultValue={currentResolution}
      source="resolution"
      translateChoice={false}
      choices={resChoices}
      onChange={(event) => {
        dispatch(setMobileResolution(event.target.value))
        window.location.reload()
      }}
    />
  )
}

const SelectDefaultView = (props) => {
  const translate = useTranslate()
  const current = localStorage.getItem('defaultView') || defaultAlbumList
  const choices = Object.keys(albumLists).map((type) => ({
    id: type,
    name: translate(`resources.album.lists.${type}`),
  }))

  return (
    <SelectInput
      {...props}
      source="defaultView"
      label={translate('menu.personal.options.defaultView')}
      defaultValue={current}
      choices={choices}
      translateChoice={false}
      onChange={(event) => {
        localStorage.setItem('defaultView', event.target.value)
      }}
    />
  )
}

const NotificationsToggle = () => {
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

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('menu.personal.name')} />
      <SimpleForm toolbar={null} variant={'outlined'}>
        <SelectTheme />
        <SelectLanguage />
        <SelectDefaultView />
        <SelectMobilePlayerResolution />
        <NotificationsToggle />
      </SimpleForm>
    </Card>
  )
}

export default Personal
