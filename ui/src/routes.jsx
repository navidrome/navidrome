import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import UserRatings from './userRatings/UserRatings'
import UserRatingItems from './userRatings/UserRatingItems'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  <Route exact path="/userRatings" render={() => <UserRatings />} key={'userRatings'} />,
  <Route
    exact
    path="/userRatings/:userId/:userName/:type/:rating"
    component={UserRatingItems}
    key={'userRatingItems'}
  />,
]

export default routes
