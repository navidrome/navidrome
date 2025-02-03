import { TableRow, TableCell } from '@material-ui/core'
import { humanize } from 'inflection'
import { useTranslate } from 'react-admin'

import en from '../i18n/en.json'
import { ArtistLinkField } from './index'

export const ParticipantsInfo = ({ classes, record }) => {
  const translate = useTranslate()
  const existingRoles = en?.resources?.artist?.roles ?? {}

  const roles = []

  if (record.participants) {
    for (const name of Object.keys(record.participants)) {
      if (name === 'albumartist' || name === 'artist') {
        continue
      }
      roles.push([name, record.participants[name].length])
    }
  }

  if (roles.length === 0) {
    return null
  }

  return (
    <>
      {roles.length > 0 && (
        <TableRow key={`${record.id}-separator`}>
          <TableCell scope="row" className={classes.tableCell}></TableCell>
          <TableCell align="left">
            <h4>{translate(`resources.song.fields.participants`)}</h4>
          </TableCell>
        </TableRow>
      )}
      {roles.map(([role, count]) => (
        <TableRow key={`${record.id}-${role}`}>
          <TableCell scope="row" className={classes.tableCell}>
            {role in existingRoles
              ? translate(`resources.artist.roles.${role}`, {
                  smart_count: count,
                })
              : humanize(role)}
            :
          </TableCell>
          <TableCell align="left">
            <ArtistLinkField source={role} record={record} limit={Infinity} />
          </TableCell>
        </TableRow>
      ))}
    </>
  )
}
