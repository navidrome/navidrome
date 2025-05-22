import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import LogViewer from './personal/LogViewer'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  <Route
    exact
    path="/personal/logs"
    render={() => <LogViewer />}
    key={'logs'}
  />,
]

export default routes
