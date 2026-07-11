import React, { useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  CardActions,
  CardContent,
  Checkbox,
  Chip,
  Divider,
  FormControl,
  FormControlLabel,
  InputLabel,
  List,
  ListItem,
  ListItemSecondaryAction,
  ListItemText,
  MenuItem,
  Select,
  Step,
  StepLabel,
  Stepper,
  TextField,
  Typography,
} from '@material-ui/core'
import ArrowDownwardIcon from '@material-ui/icons/ArrowDownward'
import ArrowUpwardIcon from '@material-ui/icons/ArrowUpward'
import DeleteIcon from '@material-ui/icons/Delete'
import EditIcon from '@material-ui/icons/Edit'
import { useNotify } from 'react-admin'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

const emptySource = {
  name: '',
  enabled: true,
  url: 'ldap://ldap.example.org:389',
  startTLS: false,
  insecureSkipVerify: false,
  bindDN: '',
  bindPassword: '',
  userBaseDN: '',
  userFilter: '(%s=%s)',
  userNameAttribute: 'uid',
  displayNameAttribute: 'cn',
  emailAttribute: 'mail',
  groupBaseDN: '',
  groupFilter: '(|(objectClass=groupOfNames)(objectClass=group))',
  groupNameAttribute: 'cn',
  groupMemberAttribute: 'member',
  requiredGroupDNs: [],
  adminGroupDNs: [],
  directBindDNTemplate: '',
  cache: { users: [], groups: [] },
}

const steps = ['Server', 'Bind & filters', 'Fetch users/groups', 'Map access']

const groupLabel = (group) => group.name || group.dn

const uniqueByDN = (groups = []) => {
  const seen = new Set()
  return groups.filter((group) => {
    if (!group.dn || seen.has(group.dn)) {
      return false
    }
    seen.add(group.dn)
    return true
  })
}

const textField = (source, setSource, key, label, props = {}) => (
  <TextField
    label={label}
    value={source[key] || ''}
    onChange={(event) => setSource({ ...source, [key]: event.target.value })}
    fullWidth
    margin="normal"
    variant="outlined"
    {...props}
  />
)

const groupSelect = (source, setSource, key, label, groups) => (
  <FormControl fullWidth margin="normal" variant="outlined">
    <InputLabel>{label}</InputLabel>
    <Select
      multiple
      value={source[key] || []}
      onChange={(event) => setSource({ ...source, [key]: event.target.value })}
      renderValue={(selected) => (
        <div>
          {selected.map((dn) => {
            const group = groups.find((candidate) => candidate.dn === dn)
            return (
              <Chip
                key={dn}
                label={group ? groupLabel(group) : dn}
                size="small"
                style={{ margin: 2 }}
              />
            )
          })}
        </div>
      )}
      label={label}
    >
      {groups.map((group) => (
        <MenuItem key={group.dn} value={group.dn}>
          <Checkbox checked={(source[key] || []).indexOf(group.dn) > -1} />
          <ListItemText primary={groupLabel(group)} secondary={group.dn} />
        </MenuItem>
      ))}
    </Select>
  </FormControl>
)

const SourceWizard = ({ initialSource, onCancel, onSave, onTest, testing }) => {
  const notify = useNotify()
  const [activeStep, setActiveStep] = useState(0)
  const [source, setSource] = useState({ ...emptySource, ...initialSource })
  const groups = useMemo(() => uniqueByDN(source.cache?.groups), [source.cache])

  const validateStep = () => {
    if (activeStep === 0 && (!source.name || !source.url)) {
      notify('LDAP name and URL are required', 'warning')
      return false
    }
    if (activeStep === 1 && (!source.userBaseDN || !source.userNameAttribute)) {
      notify('User base DN and username attribute are required', 'warning')
      return false
    }
    return true
  }

  const next = () => {
    if (validateStep()) {
      setActiveStep(activeStep + 1)
    }
  }

  const testSource = () => {
    onTest(source).then((testedSource) => {
      setSource({ ...source, ...testedSource })
      setActiveStep(3)
    })
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h5" gutterBottom>
          {source.id ? 'Edit LDAP Server' : 'Add LDAP Server'}
        </Typography>
        <Stepper activeStep={activeStep} alternativeLabel>
          {steps.map((label) => (
            <Step key={label}>
              <StepLabel>{label}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {activeStep === 0 && (
          <>
            {textField(source, setSource, 'name', 'Display name')}
            {textField(source, setSource, 'url', 'LDAP URL', {
              helperText:
                'Example: ldap://ldap.example.org:389 or ldaps://ldap.example.org:636',
            })}
            <FormControlLabel
              control={
                <Checkbox
                  checked={!!source.enabled}
                  onChange={(event) =>
                    setSource({ ...source, enabled: event.target.checked })
                  }
                />
              }
              label="Enabled"
            />
            <FormControlLabel
              control={
                <Checkbox
                  checked={!!source.startTLS}
                  onChange={(event) =>
                    setSource({ ...source, startTLS: event.target.checked })
                  }
                />
              }
              label="Use StartTLS"
            />
            <FormControlLabel
              control={
                <Checkbox
                  checked={!!source.insecureSkipVerify}
                  onChange={(event) =>
                    setSource({
                      ...source,
                      insecureSkipVerify: event.target.checked,
                    })
                  }
                />
              }
              label="Skip TLS certificate verification"
            />
          </>
        )}

        {activeStep === 1 && (
          <>
            <Typography variant="subtitle1">Service account bind</Typography>
            {textField(source, setSource, 'bindDN', 'Bind DN', {
              helperText:
                'Leave blank for anonymous bind if your LDAP server allows it.',
            })}
            {textField(source, setSource, 'bindPassword', 'Bind password', {
              type: 'password',
            })}
            {textField(
              source,
              setSource,
              'directBindDNTemplate',
              'Direct user bind DN template',
              {
                helperText:
                  'Optional. Example: uid=%s,ou=users,dc=example,dc=org. Service-account search bind is preferred.',
              },
            )}
            <Divider style={{ margin: '16px 0' }} />
            <Typography variant="subtitle1">Users</Typography>
            {textField(source, setSource, 'userBaseDN', 'User base DN')}
            {textField(source, setSource, 'userFilter', 'User filter', {
              helperText:
                'Use %s placeholders for attribute and escaped username, e.g. (%s=%s).',
            })}
            {textField(
              source,
              setSource,
              'userNameAttribute',
              'Username attribute',
            )}
            {textField(
              source,
              setSource,
              'displayNameAttribute',
              'Display name attribute',
            )}
            {textField(source, setSource, 'emailAttribute', 'Email attribute')}
            <Divider style={{ margin: '16px 0' }} />
            <Typography variant="subtitle1">Groups</Typography>
            {textField(source, setSource, 'groupBaseDN', 'Group base DN')}
            {textField(source, setSource, 'groupFilter', 'Group filter')}
            {textField(
              source,
              setSource,
              'groupNameAttribute',
              'Group name attribute',
            )}
            {textField(
              source,
              setSource,
              'groupMemberAttribute',
              'Group member attribute',
              {
                helperText:
                  'Use member for OpenLDAP groupOfNames. FreeIPA memberOf is collected from user entries automatically.',
              },
            )}
          </>
        )}

        {activeStep === 2 && (
          <>
            <Typography variant="body1" gutterBottom>
              Test the LDAP connection and service-account bind, then fetch
              users, groups, and memberships for interactive mapping.
            </Typography>
            <Button
              color="primary"
              variant="contained"
              onClick={testSource}
              disabled={testing}
            >
              Test connection and fetch directory data
            </Button>
            <Typography variant="body2" style={{ marginTop: 16 }}>
              Cached users: {source.cache?.users?.length || 0} · Cached groups:{' '}
              {source.cache?.groups?.length || 0}
            </Typography>
          </>
        )}

        {activeStep === 3 && (
          <>
            <Typography variant="body2" gutterBottom>
              Select the groups that are allowed to log in. Admin groups also
              grant Navidrome administrator access.
            </Typography>
            {groups.length === 0 ? (
              <Typography color="textSecondary">
                No groups have been discovered yet. Go back and fetch directory
                data before mapping groups.
              </Typography>
            ) : (
              <>
                {groupSelect(
                  source,
                  setSource,
                  'requiredGroupDNs',
                  'Allowed login groups',
                  groups,
                )}
                {groupSelect(
                  source,
                  setSource,
                  'adminGroupDNs',
                  'Admin groups',
                  groups,
                )}
              </>
            )}
            <Typography variant="body2" style={{ marginTop: 16 }}>
              Preview: {source.cache?.users?.length || 0} users and{' '}
              {source.cache?.groups?.length || 0} groups cached for this source.
            </Typography>
          </>
        )}
      </CardContent>
      <CardActions>
        <Button onClick={onCancel}>Cancel</Button>
        {activeStep > 0 && (
          <Button onClick={() => setActiveStep(activeStep - 1)}>Back</Button>
        )}
        {activeStep < steps.length - 1 ? (
          <Button color="primary" variant="contained" onClick={next}>
            Next
          </Button>
        ) : (
          <Button
            color="primary"
            variant="contained"
            onClick={() => onSave(source)}
          >
            Save LDAP Server
          </Button>
        )}
      </CardActions>
    </Card>
  )
}

export const LdapList = () => {
  const notify = useNotify()
  const [sources, setSources] = useState([])
  const [editingIndex, setEditingIndex] = useState(null)
  const [loading, setLoading] = useState(false)
  const [testing, setTesting] = useState(false)

  const load = () => {
    setLoading(true)
    httpClient(`${REST_URL}/ldap`)
      .then(({ json }) => setSources(json.sources || []))
      .catch(() => notify('Could not load LDAP configuration', 'warning'))
      .finally(() => setLoading(false))
  }

  useEffect(load, [notify])

  const saveSources = (nextSources) => {
    setLoading(true)
    return httpClient(`${REST_URL}/ldap`, {
      method: 'PUT',
      body: JSON.stringify({ sources: nextSources }),
    })
      .then(({ json }) => {
        setSources(json.sources || nextSources)
        notify('LDAP configuration saved')
      })
      .catch((e) => {
        notify(`Could not save LDAP configuration: ${e.message}`, 'warning')
        throw e
      })
      .finally(() => setLoading(false))
  }

  const saveSource = (source) => {
    const nextSources = [...sources]
    if (editingIndex === 'new') {
      nextSources.push(source)
    } else {
      nextSources[editingIndex] = source
    }
    saveSources(nextSources).then(() => setEditingIndex(null))
  }

  const moveSource = (index, direction) => {
    const target = index + direction
    if (target < 0 || target >= sources.length) {
      return
    }
    const nextSources = [...sources]
    const movedSource = nextSources[index]
    nextSources[index] = nextSources[target]
    nextSources[target] = movedSource
    saveSources(nextSources)
  }

  const deleteSource = (index) => {
    const nextSources = sources.filter(
      (_, sourceIndex) => sourceIndex !== index,
    )
    saveSources(nextSources)
  }

  const testSource = (source) => {
    setTesting(true)
    return httpClient(`${REST_URL}/ldap/test`, {
      method: 'POST',
      body: JSON.stringify(source),
    })
      .then(({ json }) => {
        notify(
          `LDAP test found ${json.cache?.users?.length || 0} users and ${json.cache?.groups?.length || 0} groups`,
        )
        return json
      })
      .catch((e) => {
        notify(`LDAP test failed: ${e.message}`, 'warning')
        throw e
      })
      .finally(() => setTesting(false))
  }

  if (editingIndex !== null) {
    return (
      <SourceWizard
        initialSource={
          editingIndex === 'new' ? emptySource : sources[editingIndex]
        }
        onCancel={() => setEditingIndex(null)}
        onSave={saveSource}
        onTest={testSource}
        testing={testing}
      />
    )
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h5" gutterBottom>
          LDAP Authentication
        </Typography>
        <Typography variant="body2" gutterBottom>
          LDAP sources are tried in the order shown below after internal auth
          for external clients. Use the arrow buttons to change fallback
          priority.
        </Typography>
        {sources.length === 0 ? (
          <Typography color="textSecondary" style={{ marginTop: 16 }}>
            No LDAP servers configured yet.
          </Typography>
        ) : (
          <List>
            {sources.map((source, index) => (
              <ListItem key={source.id || source.name || index} divider>
                <ListItemText
                  primary={`${index + 1}. ${source.name || 'Unnamed LDAP server'}`}
                  secondary={`${source.enabled ? 'Enabled' : 'Disabled'} · ${source.url} · ${source.cache?.users?.length || 0} cached users · ${source.cache?.groups?.length || 0} cached groups`}
                />
                <ListItemSecondaryAction>
                  <Button
                    size="small"
                    onClick={() => moveSource(index, -1)}
                    disabled={loading || index === 0}
                    startIcon={<ArrowUpwardIcon />}
                  >
                    Up
                  </Button>
                  <Button
                    size="small"
                    onClick={() => moveSource(index, 1)}
                    disabled={loading || index === sources.length - 1}
                    startIcon={<ArrowDownwardIcon />}
                  >
                    Down
                  </Button>
                  <Button
                    size="small"
                    onClick={() => setEditingIndex(index)}
                    disabled={loading}
                    startIcon={<EditIcon />}
                  >
                    Edit
                  </Button>
                  <Button
                    size="small"
                    onClick={() => deleteSource(index)}
                    disabled={loading}
                    startIcon={<DeleteIcon />}
                  >
                    Delete
                  </Button>
                </ListItemSecondaryAction>
              </ListItem>
            ))}
          </List>
        )}
      </CardContent>
      <CardActions>
        <Button
          color="primary"
          variant="contained"
          onClick={() => setEditingIndex('new')}
          disabled={loading}
        >
          + Add LDAP Server
        </Button>
      </CardActions>
    </Card>
  )
}
