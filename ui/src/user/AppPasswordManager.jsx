import React, { useCallback, useEffect, useState } from 'react'
import PropTypes from 'prop-types'
import {
  Button,
  Card,
  CardActions,
  CardContent,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@material-ui/core'
import DeleteIcon from '@material-ui/icons/Delete'
import FileCopyIcon from '@material-ui/icons/FileCopy'
import { useNotify } from 'react-admin'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

// AppPasswordManager renders a simple per-user table of long-lived app
// passwords used for Subsonic clients that cannot speak OIDC. The plaintext
// secret is shown exactly once on creation; afterwards only metadata
// (created/last used/expires) is visible.
const AppPasswordManager = ({ userId }) => {
  const notify = useNotify()
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newExpiresAt, setNewExpiresAt] = useState('')
  const [createdSecret, setCreatedSecret] = useState(null)

  const baseURL = `${REST_URL}/user/${userId}/app-password`

  const refresh = useCallback(() => {
    setLoading(true)
    httpClient(baseURL)
      .then((response) => {
        const data = response.json
        setRows(Array.isArray(data) ? data : [])
      })
      .catch((error) => notify(error.message || 'Failed to load app passwords', 'warning'))
      .finally(() => setLoading(false))
  }, [baseURL, notify])

  useEffect(() => {
    refresh()
  }, [refresh])

  const handleCreate = () => {
    const body = { name: newName }
    if (newExpiresAt) {
      body.expiresAt = new Date(newExpiresAt).toISOString()
    }
    httpClient(baseURL, { method: 'POST', body: JSON.stringify(body) })
      .then((response) => {
        setCreatedSecret(response.json)
        setNewName('')
        setNewExpiresAt('')
        setCreateOpen(false)
        refresh()
      })
      .catch((error) =>
        notify(error.message || 'Failed to create app password', 'warning'),
      )
  }

  const handleDelete = (id) => {
    if (!window.confirm('Delete this app password? Clients using it will stop working.')) {
      return
    }
    httpClient(`${baseURL}/${id}`, { method: 'DELETE' })
      .then(() => refresh())
      .catch((error) =>
        notify(error.message || 'Failed to delete app password', 'warning'),
      )
  }

  const copySecret = () => {
    if (!createdSecret?.secret) return
    navigator.clipboard
      ?.writeText(createdSecret.secret)
      .then(() => notify('Secret copied to clipboard', 'info'))
      .catch(() => notify('Could not copy to clipboard', 'warning'))
  }

  return (
    <Card style={{ marginTop: 24 }}>
      <CardContent>
        <Typography variant="h6">App passwords (Subsonic clients)</Typography>
        <Typography variant="body2" color="textSecondary">
          Generate a dedicated password for each Subsonic-compatible app. The
          secret is shown only once.
        </Typography>
        <Table size="small" style={{ marginTop: 16 }}>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Created</TableCell>
              <TableCell>Last used</TableCell>
              <TableCell>Expires</TableCell>
              <TableCell />
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.id}>
                <TableCell>{row.name}</TableCell>
                <TableCell>
                  {row.createdAt ? new Date(row.createdAt).toLocaleString() : ''}
                </TableCell>
                <TableCell>
                  {row.lastUsedAt
                    ? new Date(row.lastUsedAt).toLocaleString()
                    : '—'}
                </TableCell>
                <TableCell>
                  {row.expiresAt
                    ? new Date(row.expiresAt).toLocaleString()
                    : 'Never'}
                </TableCell>
                <TableCell align="right">
                  <Tooltip title="Delete">
                    <IconButton size="small" onClick={() => handleDelete(row.id)}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </TableCell>
              </TableRow>
            ))}
            {!loading && rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  <Typography variant="body2" color="textSecondary">
                    No app passwords yet.
                  </Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
      <CardActions>
        <Button color="primary" onClick={() => setCreateOpen(true)}>
          Generate new
        </Button>
      </CardActions>

      <Dialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>New app password</DialogTitle>
        <DialogContent>
          <TextField
            label="Name"
            fullWidth
            autoFocus
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            helperText="Friendly label, e.g. 'DSub on phone'"
          />
          <TextField
            label="Expires"
            type="datetime-local"
            fullWidth
            value={newExpiresAt}
            onChange={(e) => setNewExpiresAt(e.target.value)}
            InputLabelProps={{ shrink: true }}
            helperText="Leave blank for no expiry"
            style={{ marginTop: 16 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button color="primary" disabled={!newName} onClick={handleCreate}>
            Generate
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog
        open={createdSecret !== null}
        onClose={() => setCreatedSecret(null)}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle>Copy this secret now</DialogTitle>
        <DialogContent>
          <DialogContentText>
            This secret is shown only once. Configure your Subsonic client with
            this username and the secret below — Navidrome cannot retrieve it
            again.
          </DialogContentText>
          {createdSecret && (
            <TextField
              fullWidth
              variant="outlined"
              value={createdSecret.secret || ''}
              InputProps={{
                readOnly: true,
                endAdornment: (
                  <IconButton onClick={copySecret} size="small">
                    <FileCopyIcon fontSize="small" />
                  </IconButton>
                ),
              }}
              style={{ marginTop: 16 }}
            />
          )}
        </DialogContent>
        <DialogActions>
          <Button color="primary" onClick={() => setCreatedSecret(null)}>
            Done
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  )
}

AppPasswordManager.propTypes = {
  userId: PropTypes.string.isRequired,
}

export default AppPasswordManager
