import React from 'react'
import { TopToolbar, ExportButton } from 'react-admin'
import DeleteMissingFilesButton from './DeleteMissingFilesButton.jsx'

const MissingListActions = (props) => (
  <TopToolbar {...props}>
    <ExportButton />
    <DeleteMissingFilesButton deleteAll />
  </TopToolbar>
)

export default MissingListActions
