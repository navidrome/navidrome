import React from 'react'
import { IconButton, Tooltip } from '@material-ui/core'
import EditIcon from '@material-ui/icons/Edit'
import { useTranslate } from 'react-admin'
import { useSongEditor } from './SongEditorContext'

export const SongEditButton = ({ record }) => {
  const { openEditor } = useSongEditor()
  const translate = useTranslate()

  const handleClick = (e) => {
    e.stopPropagation()
    openEditor(record)
  }

  return (
    <Tooltip title={translate('ra.action.edit', { _: 'Edit' })}>
      <IconButton size="small" onClick={handleClick}>
        <EditIcon fontSize="small" />
      </IconButton>
    </Tooltip>
  )
}

export default SongEditButton