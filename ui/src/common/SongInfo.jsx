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
import {
  ArtistLinkField,
  BitrateField,
  ParticipantsInfo,
  PathField,
  SizeField,
} from './index'
import { MultiLineTextField } from './MultiLineTextField'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'
import { AlbumLinkField } from '../song/AlbumLinkField'

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

  // These are already displayed in other fields or are album-level tags
  const excludedTags = [
    'genre',
    'disctotal',
    'tracktotal',
    'releasetype',
    'recordlabel',
    'media',
    'albumversion',
  ]
  const data = {
    path: <PathField />,
    album: (
      <AlbumLinkField source="album" sortByOrder={'ASC'} record={record} />
    ),
    discSubtitle: <TextField source="discSubtitle" />,
    albumArtist: (
      <ArtistLinkField source="albumArtist" record={record} limit={Infinity} />
    ),
    artist: (
      <ArtistLinkField source="artist" record={record} limit={Infinity} />
    ),
    genre: (
      <FunctionField render={(r) => r.genres?.map((g) => g.name).join(' • ')} />
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

  const roles = []

  for (const name of Object.keys(record.participants)) {
    if (name === 'albumartist' || name === 'artist') {
      continue
    }
    roles.push([name, record.participants[name].length])
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

  const tags = Object.entries(record.tags ?? {}).filter(
    (tag) => !excludedTags.includes(tag[0]),
  )

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
          <ParticipantsInfo classes={classes} record={record} />
          {tags.length > 0 && (
            <TableRow key={`${record.id}-separator`}>
              <TableCell scope="row" className={classes.tableCell}></TableCell>
              <TableCell align="left">
                <h4>{translate(`resources.song.fields.tags`)}</h4>
              </TableCell>
            </TableRow>
          )}
          {tags.map(([name, values]) => (
            <TableRow key={`${record.id}-tag-${name}`}>
              <TableCell scope="row" className={classes.tableCell}>
                {name}:
              </TableCell>
              <TableCell align="left">{values.join(' • ')}</TableCell>
            </TableRow>
          ))}
          {record.rawTags && (
            <>
              <TableRow key={`${record.id}-raw-header`}>
                <TableCell
                  scope="row"
                  className={classes.tableCell}
                ></TableCell>
                <TableCell align="left">
                  <h4>{translate(`resources.song.fields.rawTags`)}</h4>
                </TableCell>
              </TableRow>
              {Object.entries(record.rawTags).map(([key, value]) => (
                <TableRow key={`${record.id}-${key}`}>
                  <TableCell scope="row" className={classes.tableCell}>
                    {key}:
                  </TableCell>
                  <TableCell align="left">{value.join(' • ')}</TableCell>
                </TableRow>
              ))}
            </>
          )}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
