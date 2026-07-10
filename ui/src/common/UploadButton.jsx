import React, { useState } from 'react'
import { CircularProgress, Tooltip, Snackbar, Box } from '@material-ui/core'
import Alert from '@material-ui/lab/Alert'
import UploadIcon from '@material-ui/icons/Publish'
import { Button } from 'react-admin'

export const UploadButton = () => {
  const [uploading, setUploading] = useState(false)
  const [toast, setToast] = useState({
    open: false,
    message: '',
    severity: 'success',
  })

  const handleFileChange = async (event) => {
    const files = event.target.files
    if (!files || files.length === 0) return

    const file = files[0]
    const formData = new FormData()
    formData.append('file', file)

    setUploading(true)

    const token =
      localStorage.getItem('x-nd-authorization') ||
      localStorage.getItem('token')

    try {
      const response = await fetch('/api/song/upload', {
        method: 'POST',
        headers: {
          ...(token && { 'X-ND-Authorization': `Bearer ${token}` }),
          ...(token && { Authorization: `Bearer ${token}` }),
        },
        body: formData,
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(errorText || 'Upload failed')
      }

      setToast({
        open: true,
        message: `"${file.name}" uploaded and scanned successfully!`,
        severity: 'success',
      })
    } catch (error) {
      let errorMsg = error.message
      try {
        const parsed = JSON.parse(error.message)
        if (parsed.error) errorMsg = parsed.error
      } catch (e) {
        void 0
      }

      setToast({
        open: true,
        message: `Error : ${errorMsg}`,
        severity: 'error',
      })
    } finally {
      setUploading(false)
      event.target.value = ''
    }
  }

  return (
    <Box display="inline-block">
      <input
        accept="audio/mp3,audio/*"
        style={{ display: 'none' }}
        id="navidrome-upload-input"
        type="file"
        onChange={handleFileChange}
        disabled={uploading}
      />
      <label htmlFor="navidrome-upload-input">
        <Tooltip title="Upload audio file">
          <span>
            {}
            <Button
              component="span"
              disabled={uploading}
              label={uploading ? 'Uploading...' : 'Uploader'}
            >
              {uploading ? (
                <CircularProgress size={18} color="inherit" />
              ) : (
                <UploadIcon />
              )}
            </Button>
          </span>
        </Tooltip>
      </label>

      <Snackbar
        open={toast.open}
        autoHideDuration={4000}
        onClose={() => setToast({ ...toast, open: false })}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert
          severity={toast.severity}
          onClose={() => setToast({ ...toast, open: false })}
          variant="filled"
        >
          {toast.message}
        </Alert>
      </Snackbar>
    </Box>
  )
}
