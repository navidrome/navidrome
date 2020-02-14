import React from 'react'
import { Admin, resolveBrowserLocale, Resource } from 'react-admin'
import dataProvider from './dataProvider'
import authProvider from './authProvider'
import polyglotI18nProvider from 'ra-i18n-polyglot'
import messages from './i18n'
import { DarkTheme, Layout, Login } from './layout'
import user from './user'
import song from './song'
import album from './album'
import artist from './artist'
import { createMuiTheme } from '@material-ui/core/styles'
import { Player, playQueueReducer } from './player'

const theme = createMuiTheme(DarkTheme)

const i18nProvider = polyglotI18nProvider(
  (locale) => (messages[locale] ? messages[locale] : messages.en),
  resolveBrowserLocale()
)

const App = () => {
  try {
    const appConfig = JSON.parse(window.__APP_CONFIG__)

    // This flags to the login process that it should create the first account instead
    if (appConfig.firstTime) {
      localStorage.setItem('initialAccountCreation', 'true')
    }
  } catch (e) {}

  return (
    <Admin
      theme={theme}
      customReducers={{ queue: playQueueReducer }}
      dataProvider={dataProvider}
      authProvider={authProvider}
      i18nProvider={i18nProvider}
      layout={Layout}
      loginPage={Login}
    >
      {(permissions) => [
        <Resource name="artist" {...artist} options={{ subMenu: 'library' }} />,
        <Resource name="album" {...album} options={{ subMenu: 'library' }} />,
        <Resource name="song" {...song} options={{ subMenu: 'library' }} />,
        <Resource name="albumSong" />,
        permissions === 'admin' ? <Resource name="user" {...user} /> : null,
        <Player />
      ]}
    </Admin>
  )
}

export default App
