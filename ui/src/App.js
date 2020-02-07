import React from 'react'
import { Admin, Resource, resolveBrowserLocale } from 'react-admin'
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
  <>
    <div>
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
          <Resource
            name="artist"
            {...artist}
            options={{ subMenu: 'library' }}
          />,
          <Resource name="album" {...album} options={{ subMenu: 'library' }} />,
          <Resource name="song" {...song} options={{ subMenu: 'library' }} />,
          permissions === 'admin' ? <Resource name="user" {...user} /> : null,
          <Player />
        ]}
      </Admin>
    </div>
  </>
)
export default App
