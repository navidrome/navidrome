import React, { useEffect, useState } from 'react'
import {
  Button,
  Card,
  CardContent,
  TextField,
  Typography,
} from '@material-ui/core'
import SettingsEthernetIcon from '@material-ui/icons/SettingsEthernet'
import { useNotify } from 'react-admin'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

const defaultConfig = { sources: [] }

export const LdapList = () => {
  const notify = useNotify()
  const [text, setText] = useState(JSON.stringify(defaultConfig, null, 2))
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    httpClient(`${REST_URL}/ldap`)
      .then(({ json }) =>
        setText(JSON.stringify({ sources: json.sources || [] }, null, 2)),
      )
      .catch(() => notify('Could not load LDAP configuration', 'warning'))
  }, [notify])

  const save = () => {
    setLoading(true)
    httpClient(`${REST_URL}/ldap`, { method: 'PUT', body: text })
      .then(({ json }) => {
        setText(JSON.stringify({ sources: json.sources || [] }, null, 2))
        notify('LDAP configuration saved')
      })
      .catch((e) =>
        notify(`Could not save LDAP configuration: ${e.message}`, 'warning'),
      )
      .finally(() => setLoading(false))
  }

  const test = () => {
    const cfg = JSON.parse(text)
    const source = cfg.sources?.[0]
    if (!source) {
      notify('Add at least one LDAP source to test', 'warning')
      return
    }
    setLoading(true)
    httpClient(`${REST_URL}/ldap/test`, {
      method: 'POST',
      body: JSON.stringify(source),
    })
      .then(({ json }) =>
        notify(
          `LDAP test found ${json.cache?.users?.length || 0} users and ${json.cache?.groups?.length || 0} groups`,
        ),
      )
      .catch((e) => notify(`LDAP test failed: ${e.message}`, 'warning'))
      .finally(() => setLoading(false))
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h5" gutterBottom>
          LDAP Authentication
        </Typography>
        <Typography variant="body2" gutterBottom>
          Configure LDAP sources as JSON. Sources are evaluated after internal
          auth for external clients; the login page shows enabled sources as
          tabs.
        </Typography>
        <TextField
          multiline
          minRows={24}
          fullWidth
          variant="outlined"
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <Button
          color="primary"
          variant="contained"
          onClick={save}
          disabled={loading}
          style={{ marginTop: 16, marginRight: 8 }}
        >
          Save
        </Button>
        <Button
          variant="outlined"
          onClick={test}
          disabled={loading}
          style={{ marginTop: 16 }}
        >
          Test first source
        </Button>
      </CardContent>
    </Card>
  )
}
