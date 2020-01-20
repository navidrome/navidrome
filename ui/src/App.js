// in src/App.js
import React from 'react'
import { Admin, Resource } from 'react-admin'
import dataProvider from './dataProvider'
import authProvider from './authProvider'
import { Login } from './layout'
import user from './user'

const App = () => (
  <Admin
    dataProvider={dataProvider}
    authProvider={authProvider}
    loginPage={Login}
  >
    <Resource name="user" {...user} />
  </Admin>
)
export default App
