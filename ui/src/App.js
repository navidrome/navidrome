import React from 'react'
import ReactGA from 'react-ga'
import 'react-jinke-music-player/assets/index.css'
import { Provider, useDispatch } from 'react-redux'
import { createHashHistory } from 'history'
import { Admin as RAAdmin, Resource } from 'react-admin'
import dataProvider from './dataProvider'
import authProvider from './authProvider'
import { Layout, Login, Logout } from './layout'
import transcoding from './transcoding'
import player from './player'
import user from './user'
import song from './song'
import album from './album'
import artist from './artist'
import playlist from './playlist'
import { Player } from './audioplayer'
import customRoutes from './routes'
import {
  themeReducer,
  addToPlaylistDialogReducer,
  playQueueReducer,
  albumViewReducer,
  activityReducer,
  settingsReducer,
} from './reducers'
import createAdminStore from './store/createAdminStore'
import { i18nProvider } from './i18n'
import config from './config'
import { setDispatch, startEventStream } from './eventStream'
import { HotKeys } from 'react-hotkeys'
import { keyMap } from './hotkeys'

const history = createHashHistory()

if (config.gaTrackingId) {
  ReactGA.initialize(config.gaTrackingId)
  history.listen((location) => {
    ReactGA.pageview(location.pathname)
  })
  ReactGA.pageview(window.location.pathname)
}

const App = () => (
  <Provider
    store={createAdminStore({
      authProvider,
      dataProvider,
      history,
      customReducers: {
        queue: playQueueReducer,
        albumView: albumViewReducer,
        theme: themeReducer,
        addToPlaylistDialog: addToPlaylistDialogReducer,
        activity: activityReducer,
        settings: settingsReducer,
      },
    })}
  >
    <Admin />
  </Provider>
)

const Admin = (props) => {
  const dispatch = useDispatch()
  if (config.devActivityPanel) {
    setDispatch(dispatch)
    authProvider
      .checkAuth()
      .then(() => startEventStream())
      .catch(() => {}) // ignore if not logged in
  }

  return (
    <RAAdmin
      disableTelemetry
      dataProvider={dataProvider}
      authProvider={authProvider}
      i18nProvider={i18nProvider}
      customRoutes={customRoutes}
      history={history}
      layout={Layout}
      loginPage={Login}
      logoutButton={Logout}
      {...props}
    >
      {(permissions) => [
        <Resource name="album" {...album} options={{ subMenu: 'albumList' }} />,
        <Resource name="artist" {...artist} options={{ subMenu: 'library' }} />,
        <Resource name="song" {...song} options={{ subMenu: 'library' }} />,
        <Resource
          name="playlist"
          {...playlist}
          options={{ subMenu: 'playlists' }}
        />,
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
        <Resource name="albumSong" />,
        <Resource name="translation" />,
        <Resource name="playlistTrack" />,
        <Resource name="keepalive" />,

        <Player />,
      ]}
    </RAAdmin>
  )
}

const AppWithHotkeys = () => (
  <HotKeys keyMap={keyMap}>
    <App />
  </HotKeys>
)

export default AppWithHotkeys
