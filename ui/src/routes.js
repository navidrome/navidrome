import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import ImgMediaCard from './common/ArtistDetail'
import ArtistView from './common/ArtistDetail'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} />,
  <Route
    path="/iartist/:id"
    render={(props) => (
      <>
        <ArtistView artist={props.match.params.id} />
      </>
    )}
  />,
]

export default routes
