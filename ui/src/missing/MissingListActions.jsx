import React from 'react'
import { TopToolbar, ExportButton, useListContext } from 'react-admin'
import DeleteMissingFilesButton from './DeleteMissingFilesButton.jsx'

const MissingListActions = (props) => {
  const { total } = useListContext()
  return (
    <TopToolbar {...props}>
      <ExportButton maxResults={total} />
      <DeleteMissingFilesButton deleteAll />
    </TopToolbar>
  )
}

export default MissingListActions
