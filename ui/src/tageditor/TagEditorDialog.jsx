import React from 'react'
import PropTypes from 'prop-types'
import { useNotify, useRefresh } from 'react-admin'
import {
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Switch,
  TextField,
  Button,
} from '@material-ui/core'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

const jsonHeaders = new Headers({
  Accept: 'application/json',
  'Content-Type': 'application/json',
})

const TagEditorDialog = ({
  open,
  onClose,
  title,
  endpoint,
  fields,
  saveLabel,
  onSaved,
}) => {
  const notify = useNotify()
  const refresh = useRefresh()
  const [loading, setLoading] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const [values, setValues] = React.useState({})

  React.useEffect(() => {
    if (!open) {
      return
    }

    setLoading(true)
    httpClient(`${REST_URL}${endpoint}`)
      .then(({ json }) => {
        setValues(json || {})
      })
      .catch((error) => {
        notify(
          error?.body?.error || error?.message || 'Could not load current tags',
          'warning',
        )
        onClose()
      })
      .finally(() => {
        setLoading(false)
      })
  }, [endpoint, notify, onClose, open])

  const handleFieldChange = React.useCallback(
    (name, type = 'text') =>
      (event) => {
        const value = type === 'boolean' ? event.target.checked : event.target.value
        setValues((current) => ({
          ...current,
          [name]: value,
        }))
      },
    [],
  )

  const handleSave = React.useCallback(async () => {
    setSaving(true)
    try {
      const { json } = await httpClient(`${REST_URL}${endpoint}`, {
        method: 'PUT',
        headers: jsonHeaders,
        body: JSON.stringify(values),
      })
      notify('Tags updated', 'info')
      refresh()
      onSaved && onSaved(json)
      onClose()
    } catch (error) {
      notify(
        error?.body?.error || error?.message || 'Could not save tags',
        'warning',
      )
    } finally {
      setSaving(false)
    }
  }, [endpoint, notify, onClose, onSaved, refresh, values])

  return (
    <Dialog open={open} onClose={saving ? undefined : onClose} fullWidth maxWidth="sm">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent dividers>
        {loading ? (
          <div
            style={{
              alignItems: 'center',
              display: 'flex',
              justifyContent: 'center',
              minHeight: 160,
            }}
          >
            <CircularProgress size={28} />
          </div>
        ) : (
          fields.map((field) =>
            field.type === 'boolean' ? (
              <FormControlLabel
                key={field.name}
                control={
                  <Switch
                    checked={Boolean(values[field.name])}
                    color="primary"
                    onChange={handleFieldChange(field.name, 'boolean')}
                  />
                }
                label={field.label}
                style={{ display: 'flex', marginBottom: 8 }}
              />
            ) : (
              <TextField
                key={field.name}
                fullWidth
                margin="dense"
                multiline={field.multiline}
                rows={field.rows}
                label={field.label}
                helperText={field.helperText}
                value={values[field.name] || ''}
                onChange={handleFieldChange(field.name)}
              />
            ),
          )
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button color="primary" onClick={handleSave} disabled={loading || saving}>
          {saving ? <CircularProgress size={18} /> : saveLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

TagEditorDialog.propTypes = {
  endpoint: PropTypes.string.isRequired,
  fields: PropTypes.arrayOf(PropTypes.object).isRequired,
  onClose: PropTypes.func.isRequired,
  onSaved: PropTypes.func,
  open: PropTypes.bool.isRequired,
  saveLabel: PropTypes.string,
  title: PropTypes.string.isRequired,
}

TagEditorDialog.defaultProps = {
  onSaved: null,
  saveLabel: 'Save tags',
}

export default TagEditorDialog
