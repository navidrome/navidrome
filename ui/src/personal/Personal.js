import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Card } from '@material-ui/core'
import {
  SelectInput,
  SimpleForm,
  Title,
  useLocale,
  useSetLocale,
  useTranslate,
  BooleanInput,
  useNotify,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import HelpOutlineIcon from '@material-ui/icons/HelpOutline'
import { 
  changeTheme,
  setNotificationsState,
} from '../actions'
import themes from '../themes'
import { docsUrl } from '../utils'
import { useGetLanguageChoices } from '../i18n'
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

const NotificationsToggle = (props) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const notify = useNotify()
  const currentSetting = useSelector((state) => state.settings.notifications)
  const current = (() => {
    if (!("Notification" in window) || Notification.permission !== 'granted') {
      return false
    }
    return currentSetting
  })()

  return (
    <BooleanInput
      {...props}
      source='notifications'
      label={translate('menu.personal.options.desktop_notifications')}
      defaultValue={current}
      onChange={async (notificationsEnabled) => {
        if (notificationsEnabled) {
          if (!('Notification' in window) || Notification.permission === 'denied') {
            notify(translate('message.notifications_blocked'), 'warning')
            notificationsEnabled = false
          } else {
            const response = await Notification.requestPermission()
            if (response !== 'granted') {
              notificationsEnabled = false
            }
          }
          if (!notificationsEnabled) {
            // Need to turn switch off
          }
        }
        dispatch(setNotificationsState(notificationsEnabled))
      }}
    />
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
        <NotificationsToggle />
      </SimpleForm>
    </Card>
  )
}

export default Personal
