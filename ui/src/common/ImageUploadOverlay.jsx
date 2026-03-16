import { IconButton, Tooltip } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import PhotoCameraIcon from '@material-ui/icons/PhotoCamera'
import DeleteIcon from '@material-ui/icons/Delete'
import { useTranslate, useNotify, useRefresh } from 'react-admin'
import { useCallback, useRef } from 'react'
import config from '../config'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'

const useStyles = makeStyles(() => ({
  coverOverlay: {
    position: 'absolute',
    bottom: 0,
    right: 0,
    display: 'flex',
    gap: '2px',
    padding: '2px',
    backgroundColor: 'rgba(0,0,0,0.5)',
    borderRadius: '4px 0 0 0',
    opacity: 0,
    transition: 'opacity 0.2s ease-in-out',
    '*:hover > &': {
      opacity: 1,
    },
  },
  overlayButton: {
    color: '#fff',
    padding: '4px',
    '&:hover': {
      backgroundColor: 'rgba(255,255,255,0.2)',
    },
  },
  overlayIcon: {
    fontSize: '1.2rem',
  },
}))

export const ImageUploadOverlay = ({
  entityType,
  entityId,
  hasUploadedImage,
  onImageChange,
}) => {
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const classes = useStyles()
  const fileInputRef = useRef(null)

  const canEdit =
    config.enableCoverArtUpload || localStorage.getItem('role') === 'admin'

  const handleUploadClick = useCallback((e) => {
    e.stopPropagation()
    if (fileInputRef.current) {
      fileInputRef.current.click()
    }
  }, [])

  const handleFileChange = useCallback(
    async (e) => {
      const file = e.target.files[0]
      if (!file || !entityId) return

      const formData = new FormData()
      formData.append('image', file)

      try {
        await httpClient(`${REST_URL}/${entityType}/${entityId}/image`, {
          method: 'POST',
          headers: new Headers({}),
          body: formData,
        })
        notify(`message.coverUploaded`, 'success')
        if (onImageChange) onImageChange()
        refresh()
      } catch (err) {
        notify(`message.coverUploadError`, 'warning')
      }

      e.target.value = ''
    },
    [entityType, entityId, notify, refresh, onImageChange],
  )

  const handleRemoveCover = useCallback(
    async (e) => {
      e.stopPropagation()
      if (!entityId) return

      try {
        await httpClient(`${REST_URL}/${entityType}/${entityId}/image`, {
          method: 'DELETE',
        })
        notify(`message.coverRemoved`, 'success')
        if (onImageChange) onImageChange()
        refresh()
      } catch (err) {
        notify(`message.coverRemoveError`, 'warning')
      }
    },
    [entityType, entityId, notify, refresh, onImageChange],
  )

  if (!canEdit) return null

  return (
    <div className={classes.coverOverlay}>
      <Tooltip title={translate(`message.uploadCover`)}>
        <IconButton
          className={classes.overlayButton}
          onClick={handleUploadClick}
          size="small"
        >
          <PhotoCameraIcon className={classes.overlayIcon} />
        </IconButton>
      </Tooltip>
      {hasUploadedImage && (
        <Tooltip title={translate(`message.removeCover`)}>
          <IconButton
            className={classes.overlayButton}
            onClick={handleRemoveCover}
            size="small"
          >
            <DeleteIcon className={classes.overlayIcon} />
          </IconButton>
        </Tooltip>
      )}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />
    </div>
  )
}
