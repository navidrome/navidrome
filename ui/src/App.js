import React from 'react'
import { Admin, Resource } from 'react-admin'
import dataProvider from './dataProvider'
import authProvider from './authProvider'
import { DarkTheme, Layout, Login } from './layout'
import user from './user'
import song from './song'
import album from './album'
import artist from './artist'
import { createMuiTheme } from '@material-ui/core/styles'
import { Player, playQueueReducer } from './player'

const theme = createMuiTheme(DarkTheme)

const App = () => (
  <>
    <div>
      <Admin
        theme={theme}
        customReducers={{ queue: playQueueReducer }}
        dataProvider={dataProvider}
        authProvider={authProvider}
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
