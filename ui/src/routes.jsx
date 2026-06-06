import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import StarredSongList from './song/StarredSongList'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  <Route
    exact
    path="/favourites"
    render={(props) => <StarredSongList {...props} />}
    key={'favourites'}
  />,
]

export default routes
