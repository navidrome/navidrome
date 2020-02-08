import React, { useState } from 'react'
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

const App = () => (
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
      permissions === 'admin' ? <Resource name="user" {...user} /> : null,
      <Player />
    ]}
  </Admin>
)

// TODO: This is a complicated way to force a first check for initial setup. A better way would be to send this info
// set in the `window` object in the index.html
const AppWrapper = () => {
  const [checked, setChecked] = useState(false)

  if (!checked) {
    dataProvider
      .getOne('keepalive', { id: new Date().getTime() })
      .then(() => setChecked(true))
      .catch((err) => {
        authProvider
          .checkError(err)
          .then(() => {
            setChecked(true)
          })
          .catch(() => {
            setChecked(true)
          })
      })
    return null
  }
  return <App />
}

export default AppWrapper
