import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import inflection from 'inflection'
import TableCell from '@material-ui/core/TableCell'
import TableContainer from '@material-ui/core/TableContainer'
import TableRow from '@material-ui/core/TableRow'
import {
  ArrayField,
  BooleanField,
  ChipField,
  DateField,
  FunctionField,
  SingleFieldList,
  TextField,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { MultiLineTextField } from '../common'

const useStyles = makeStyles({
  tableCell: {
    width: '17.5%',
  },
})

const AlbumInfo = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const record = useRecordContext(props)
  const data = {
    album: <TextField source={'name'} />,
    albumArtist: <TextField source={'albumArtist'} />,
    genre: (
      <ArrayField source={'genres'}>
        <SingleFieldList linkType={false}>
          <ChipField source={'name'} />
        </SingleFieldList>
      </ArrayField>
    ),
    recordLabel: (
      <FunctionField
        source={'recordLabel'}
        render={(record) => record.tags?.recordlabel?.join(', ')}
      />
    ),
    releaseType: (
      <FunctionField
        source={'releaseType'}
        render={(record) => record.tags?.releasetype?.join(', ')}
      />
    ),
    grouping: (
      <FunctionField
        source={'grouping'}
        render={(record) => record.tags?.grouping?.join(', ')}
      />
    ),
    mood: (
      <FunctionField
        source={'mood'}
        render={(record) => record.tags?.mood?.join(', ')}
      />
    ),
    compilation: <BooleanField source={'compilation'} />,
    updatedAt: <DateField source={'updatedAt'} showTime />,
    comment: <MultiLineTextField source={'comment'} />,
  }

  const optionalFields = ['comment', 'genre']
  optionalFields.forEach((field) => {
    !record[field] && delete data[field]
  })

  const optionalTags = ['releaseType', 'recordLabel', 'grouping', 'mood']
  optionalTags.forEach((field) => {
    !record?.tags?.[field.toLowerCase()] && delete data[field]
  })

  return (
    <TableContainer>
      <Table aria-label="album details" size="small">
        <TableBody>
          {Object.keys(data).map((key) => {
            return (
              <TableRow key={`${record.id}-${key}`}>
                <TableCell
                  component="th"
                  scope="row"
                  className={classes.tableCell}
                >
                  {translate(`resources.album.fields.${key}`, {
                    _: inflection.humanize(inflection.underscore(key)),
                  })}
                  :
                </TableCell>
                <TableCell align="left">{data[key]}</TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

export default AlbumInfo
