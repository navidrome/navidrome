import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import { JourneyHome } from './musicJourney'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  // Alias for the dashboard, so deep links like #/journey keep working.
  <Route
    exact
    path="/journey"
    render={() => <JourneyHome />}
    key={'journey'}
  />,
]

export default routes
