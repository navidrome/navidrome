// in src/App.js
import React from 'react'
import { Admin, Resource } from 'react-admin'
import dataProvider from './dataProvider'
import authProvider from './authProvider'
import { Login, Layout } from './layout'
import user from './user'
import song from './song'
import album from './album'

const App = () => (
  <Admin
    dataProvider={dataProvider}
    authProvider={authProvider}
    layout={Layout}
    loginPage={Login}
  >
    <Resource name="song" {...song} options={{ subMenu: 'library' }} />
    <Resource name="album" {...album} options={{ subMenu: 'library' }} />
    <Resource name="user" {...user} />
  </Admin>
)
export default App
