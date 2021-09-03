import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import ArtistView from './common/ArtistDetail'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} />,
  <Route
    path="/artist/:id"
    render={(props) => (
      <>
        <ArtistView artist={props.match.params.id} />
      </>
    )}
  />,
]

export default routes
