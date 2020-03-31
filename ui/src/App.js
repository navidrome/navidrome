import React from 'react'
import { Admin, resolveBrowserLocale, Resource } from 'react-admin'
import dataProvider from './dataProvider'
import authProvider from './authProvider'
import polyglotI18nProvider from 'ra-i18n-polyglot'
import messages from './i18n'
import { Layout, Login } from './layout'
import transcoding from './transcoding'
import player from './player'
import user from './user'
import song from './song'
import album from './album'
import artist from './artist'
import { Player, playQueueReducer } from './audioplayer'
import { albumViewReducer } from './album/albumState'
import customRoutes from './routes'
import themeReducer from './configuration/themeReducer'

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
      customReducers={{
        queue: playQueueReducer,
        albumView: albumViewReducer,
        theme: themeReducer
      }}
      dataProvider={dataProvider}
      authProvider={authProvider}
      i18nProvider={i18nProvider}
      customRoutes={customRoutes}
      layout={Layout}
      loginPage={Login}
    >
      {(permissions) => [
        <Resource name="artist" {...artist} options={{ subMenu: 'library' }} />,
        <Resource name="album" {...album} options={{ subMenu: 'library' }} />,
        <Resource name="song" {...song} options={{ subMenu: 'library' }} />,
        <Resource name="albumSong" />,
        permissions === 'admin' ? (
          <Resource name="user" {...user} options={{ subMenu: 'settings' }} />
        ) : null,
        <Resource
          name="player"
          {...player}
          options={{ subMenu: 'settings' }}
        />,
        permissions === 'admin' ? (
          <Resource
            name="transcoding"
            {...transcoding}
            options={{ subMenu: 'settings' }}
          />
        ) : (
          <Resource name="transcoding" />
        ),
        <Player />
      ]}
    </Admin>
  )
}

export default App
