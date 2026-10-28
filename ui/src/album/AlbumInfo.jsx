import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import { humanize, underscore } from 'inflection'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableRow from '@mui/material/TableRow'
import {
  ArrayField,
  BooleanField,
  ChipField,
  FunctionField,
  SingleFieldList,
  TextField,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import { useTheme } from '@mui/material/styles'
import {
  DateField,
  ArtistLinkField,
  MultiLineTextField,
  ParticipantsInfo,
  RangeField,
} from '../common'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles({
  tableCell: {
    width: '17.5%',
  },
  value: {
    whiteSpace: 'pre-line',
  },
})

const AlbumInfo = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const record = useRecordContext(props)
  const data = {
    name: <TextField source={'name'} />,
    libraryName: <TextField source="libraryName" />,
    albumArtist: (
      <ArtistLinkField source="albumArtist" record={record} limit={Infinity} />
    ),
    genre: (
      <ArrayField source={'genres'}>
        <SingleFieldList linkType={false}>
          <ChipField source={'name'} />
        </SingleFieldList>
      </ArrayField>
    ),
    date:
      record?.maxYear && record.maxYear === record.minYear ? (
        <TextField source={'date'} />
      ) : (
        <RangeField source={'year'} />
      ),
    originalDate:
      record?.maxOriginalYear &&
      record.maxOriginalYear === record.minOriginalYear ? (
        <TextField source={'originalDate'} />
      ) : (
        <RangeField source={'originalYear'} />
      ),
    releaseDate: <TextField source={'releaseDate'} />,
    recordLabel: (
      <FunctionField
        source={'recordLabel'}
        render={(record) => record.tags?.recordlabel?.join(', ')}
      />
    ),
    catalogNum: <TextField source={'catalogNum'} />,
    releaseType: (
      <FunctionField
        source={'releaseType'}
        render={(record) => record.tags?.releasetype?.join(', ')}
      />
    ),
    media: (
      <FunctionField
        source={'media'}
        render={(record) => record.tags?.media?.join(', ')}
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

  const optionalFields = ['comment', 'genre', 'catalogNum']
  optionalFields.forEach((field) => {
    !record[field] && delete data[field]
  })

  const optionalTags = [
    'releaseType',
    'recordLabel',
    'grouping',
    'mood',
    'media',
  ]
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
          <ParticipantsInfo record={record} classes={classes} />
        </TableBody>
      </Table>
    </TableContainer>
  )
}

export default AlbumInfo
