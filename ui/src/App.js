// in src/App.js
import React from 'react'
import { Admin, Resource } from 'react-admin'
import jsonServerProvider from 'ra-data-json-server'
import user from './user'

const dataProvider = jsonServerProvider('/app/api')
const App = () => (
  <Admin dataProvider={dataProvider}>
    <Resource name="user" {...user} />
  </Admin>
)
export default App
