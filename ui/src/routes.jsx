import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
]

export default routes
