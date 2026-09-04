import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import { humanize, underscore } from 'inflection'
import TableCell from '@material-ui/core/TableCell'
import TableContainer from '@material-ui/core/TableContainer'
import TableRow from '@material-ui/core/TableRow'
import {
  DateField,
  TextField,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { ArtworkInfo, SizeField } from '../common'

const useStyles = makeStyles({
  tableCell: {
    width: '17.5%',
  },
  value: {
    whiteSpace: 'pre-line',
  },
})

const ArtistInfo = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const record = useRecordContext(props)
  const data = {
    name: <TextField source={'name'} />,
    sortArtistName: <TextField source={'sortArtistName'} />,
    mbzArtistId: <TextField source={'mbzArtistId'} />,
    albumCount: <TextField source={'albumCount'} />,
    songCount: <TextField source={'songCount'} />,
    size: <SizeField source={'size'} />,
    playCount: <TextField source={'playCount'} />,
    externalInfoUpdatedAt: (
      <DateField source={'externalInfoUpdatedAt'} showTime />
    ),
    updatedAt: <DateField source={'updatedAt'} showTime />,
  }

  const optionalFields = [
    'sortArtistName',
    'mbzArtistId',
    'externalInfoUpdatedAt',
  ]
  optionalFields.forEach((field) => {
    !record[field] && delete data[field]
  })

  return (
    <TableContainer>
      <Table aria-label="artist details" size="small">
        <TableBody>
          {Object.keys(data).map((key) => {
            return (
              <TableRow key={`${record.id}-${key}`}>
                <TableCell
                  component="th"
                  scope="row"
                  className={classes.tableCell}
                >
                  {translate(`resources.artist.fields.${key}`, {
                    _: humanize(underscore(key)),
                  })}
                  :
                </TableCell>
                <TableCell align="left" className={classes.value}>
                  {data[key]}
                </TableCell>
              </TableRow>
            )
          })}
          <ArtworkInfo resource="artist" id={record.id} />
        </TableBody>
      </Table>
    </TableContainer>
  )
}

export default ArtistInfo
