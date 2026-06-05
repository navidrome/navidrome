import React from 'react'
import LibraryScanButton from './LibraryScanButton'

const LibraryListBulkActions = (props) => (
  <>
    <LibraryScanButton fullScan={false} {...props} />
    <LibraryScanButton fullScan={true} {...props} />
  </>
)

export default LibraryListBulkActions
