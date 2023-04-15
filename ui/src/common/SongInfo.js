import React from 'react'
import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import TableCell from '@material-ui/core/TableCell'
import TableContainer from '@material-ui/core/TableContainer'
import TableRow from '@material-ui/core/TableRow'
import {
  BooleanField,
  DateField,
  TextField,
  NumberField,
  FunctionField,
  useTranslate,
  useRecordContext,
} from 'react-admin'
import inflection from 'inflection'
import { BitrateField, SizeField } from './index'
import { MultiLineTextField } from './MultiLineTextField'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'

const useStyles = makeStyles({
  gain: {
    '&:after': {
      content: (props) => (props.gain ? " ' db'" : ''),
    },
  },
  tableCell: {
    width: '17.5%',
  },
})

export const SongInfo = (props) => {
  const classes = useStyles({ gain: config.enableReplayGain })
  const translate = useTranslate()
  const record = useRecordContext(props)
  const data = {
    path: <TextField source="path" />,
    album: <TextField source="album" />,
    discSubtitle: <TextField source="discSubtitle" />,
    albumArtist: <TextField source="albumArtist" />,
    genre: (
      <FunctionField render={(r) => r.genres?.map((g) => g.name).join(', ')} />
    ),
    compilation: <BooleanField source="compilation" />,
    bitRate: <BitrateField source="bitRate" />,
    channels: <NumberField source="channels" />,
    size: <SizeField source="size" />,
    updatedAt: <DateField source="updatedAt" showTime />,
    playCount: <TextField source="playCount" />,
    bpm: <NumberField source="bpm" />,
    comment: <MultiLineTextField source="comment" />,
  }

  const optionalFields = ['discSubtitle', 'comment', 'bpm', 'genre']
  optionalFields.forEach((field) => {
    !record[field] && delete data[field]
  })
  if (record.playCount > 0) {
    data.playDate = <DateField record={record} source="playDate" showTime />
  }

  if (config.enableReplayGain) {
    data.albumGain = (
      <NumberField source="rgAlbumGain" className={classes.gain} />
    )
    data.trackGain = (
      <NumberField source="rgTrackGain" className={classes.gain} />
    )
  }

  return (
    <TableContainer>
      <Table aria-label="song details" size="small">
        <TableBody>
          {Object.keys(data).map((key) => {
            return (
              <TableRow key={`${record.id}-${key}`}>
                <TableCell scope="row" className={classes.tableCell}>
                  {translate(`resources.song.fields.${key}`, {
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
