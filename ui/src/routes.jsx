import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import Stats from './stats/Stats'

const routes = [
  <Route path="/personal" element=<Personal />} key={'personal'} />,
]

export default routes
