// in src/App.js
import React from 'react'
import { Admin, Resource } from 'react-admin'
import dataProvider from './dataProvider'
import authProvider from './authProvider'
import { Login, Layout, DarkTheme } from './layout'
import user from './user'
import song from './song'
import album from './album'
import artist from './artist'
import { createMuiTheme } from '@material-ui/core/styles'

const theme = createMuiTheme(DarkTheme)

const App = () => (
  <Admin
    theme={theme}
    dataProvider={dataProvider}
    authProvider={authProvider}
    layout={Layout}
    loginPage={Login}
  >
    {(permissions) => [
      <Resource name="artist" {...artist} options={{ subMenu: 'library' }} />,
      <Resource name="album" {...album} options={{ subMenu: 'library' }} />,
      <Resource name="song" {...song} options={{ subMenu: 'library' }} />,
      permissions === 'admin' ? <Resource name="user" {...user} /> : null
    ]}
  </Admin>
)
export default App
