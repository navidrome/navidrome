import React from 'react'
import { ResourceContextProvider } from 'react-admin'
import SongList from './SongList'

// Favourite Songs view: reuses the regular SongList with a PERMANENT
// { starred: true } filter. react-admin keeps permanent filters separate from
// the user-editable (stored) filters, so opening this view never leaks the
// starred filter into the normal Songs list. Sorted by most-recently-favourited.
const StarredSongList = (props) => (
  <ResourceContextProvider value="song">
    <SongList
      {...props}
      resource="song"
      basePath="/song"
      hasCreate={false}
      hasEdit={false}
      hasShow={false}
      hasList={true}
      filter={{ starred: true }}
      sort={{ field: 'starred_at', order: 'DESC' }}
    />
  </ResourceContextProvider>
)

export default StarredSongList
