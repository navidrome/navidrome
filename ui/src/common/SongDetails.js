import React from 'react'
import Paper from '@material-ui/core/Paper'
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
  useTranslate,
} from 'react-admin'
import inflection from 'inflection'
import { BitrateField, SizeField } from './index'
import { MultiLineTextField } from './MultiLineTextField'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles({
  tableCell: {
    width: '17.5%',
  },
})

export const SongDetails = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const { record } = props
  const data = {
    path: <TextField record={record} source="path" />,
    album: <TextField record={record} source="album" />,
    discSubtitle: <TextField record={record} source="discSubtitle" />,
    albumArtist: <TextField record={record} source="albumArtist" />,
    genre: <TextField record={record} source="genre" />,
    compilation: <BooleanField record={record} source="compilation" />,
    bitRate: <BitrateField record={record} source="bitRate" />,
    size: <SizeField record={record} source="size" />,
    updatedAt: <DateField record={record} source="updatedAt" showTime />,
    playCount: <TextField record={record} source="playCount" />,
    bpm: <NumberField record={record} source="bpm" />,
    comment: <MultiLineTextField record={record} source="comment" />,
  }
  if (!record.discSubtitle) {
    delete data.discSubtitle
  }
  if (!record.comment) {
    delete data.comment
  }
  if (!record.bpm) {
    delete data.bpm
  }
  if (record.playCount > 0) {
    data.playDate = <DateField record={record} source="playDate" showTime />
  }
  return (
    <TableContainer component={Paper}>
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
