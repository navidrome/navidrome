import { TableRow, TableCell } from '@material-ui/core'
import { humanize } from 'inflection'
import { useTranslate } from 'react-admin'

import en from '../i18n/en.json'
import { ArtistLinkField } from './index'
import { InfoLabelCell } from './InfoTable'

const groupBySubRole = (role, artists) => {
  const groups = new Map()
  for (const { subRole, ...artist } of artists) {
    const key = subRole ?? ''
    if (!groups.has(key)) {
      groups.set(key, [])
    }
    groups.get(key).push(artist)
  }
  return [...groups].map(([subRole, artists]) => ({ role, subRole, artists }))
}

export const ParticipantsInfo = ({ record }) => {
  const translate = useTranslate()
  const existingRoles = en?.resources?.artist?.roles ?? {}

  const rows = []

  if (record.participants) {
    for (const [role, artists] of Object.entries(record.participants)) {
      if (role === 'albumartist' || role === 'artist') {
        continue
      }
      rows.push(...groupBySubRole(role, artists))
    }
  }

  if (rows.length === 0) {
    return null
  }

  const roleLabel = (role, count) =>
    role in existingRoles
      ? translate(`resources.artist.roles.${role}`, { smart_count: count })
      : humanize(role)

  return (
    <>
      <TableRow key={`${record.id}-separator`}>
        <InfoLabelCell />
        <TableCell align="left">
          <h4>{translate(`resources.song.fields.participants`)}</h4>
        </TableCell>
      </TableRow>
      {rows.map(({ role, subRole, artists }) => (
        <TableRow key={`${record.id}-${role}-${subRole}`}>
          <InfoLabelCell>
            {roleLabel(role, artists.length)}
            {subRole && ` (${subRole})`}:
          </InfoLabelCell>
          <TableCell align="left">
            <ArtistLinkField
              source={role}
              record={{ participants: { [role]: artists } }}
              limit={Infinity}
            />
          </TableCell>
        </TableRow>
      ))}
    </>
  )
}
