import React, { useState } from 'react'
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
import { humanize, underscore } from 'inflection'
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
import { Tab, Tabs } from '@material-ui/core'

const useStyles = makeStyles({
  gain: {
    '&:after': {
      content: (props) => (props.gain ? " ' db'" : ''),
    },
  },
  tableCell: {
    width: '17.5%',
  },
  value: {
    whiteSpace: 'pre-line',
  },
})

export const SongInfo = (props) => {
  const classes = useStyles({ gain: config.enableReplayGain })
  const translate = useTranslate()
  const record = useRecordContext(props)
  const [tab, setTab] = useState(0)

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
    libraryName: <TextField source="libraryName" />,
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
    bitDepth: <NumberField source="bitDepth" />,
    sampleRate: <NumberField source="sampleRate" />,
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

  const optionalFields = [
    'discSubtitle',
    'comment',
    'bpm',
    'genre',
    'bitDepth',
    'sampleRate',
  ]
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
      {record.rawTags && (
        <Tabs value={tab} onChange={(_, value) => setTab(value)}>
          <Tab
            label={translate(`resources.song.fields.mappedTags`)}
            id="mapped-tags-tab"
            aria-controls="mapped-tags-body"
          />
          <Tab
            label={translate(`resources.song.fields.rawTags`)}
            id="raw-tags-tab"
            aria-controls="raw-tags-body"
          />
        </Tabs>
      )}
      <div
        hidden={tab === 1}
        id="mapped-tags-body"
        aria-labelledby={record.rawTags ? 'mapped-tags-tab' : undefined}
      >
        <Table aria-label="song details" size="small">
          <TableBody>
            {Object.keys(data).map((key) => {
              return (
                <TableRow key={`${record.id}-${key}`}>
                  <TableCell scope="row" className={classes.tableCell}>
                    {translate(`resources.song.fields.${key}`, {
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
            <ParticipantsInfo classes={classes} record={record} />
            {tags.length > 0 && (
              <TableRow key={`${record.id}-separator`}>
                <TableCell
                  scope="row"
                  className={classes.tableCell}
                ></TableCell>
                <TableCell align="left" className={classes.value}>
                  <h4>{translate(`resources.song.fields.tags`)}</h4>
                </TableCell>
              </TableRow>
            )}
            {tags.map(([name, values]) => (
              <TableRow key={`${record.id}-tag-${name}`}>
                <TableCell scope="row" className={classes.tableCell}>
                  {name}:
                </TableCell>
                <TableCell align="left" className={classes.value}>
                  {values.join(' • ')}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {record.rawTags && (
        <div
          hidden={tab === 0}
          id="raw-tags-body"
          aria-labelledby="raw-tags-tab"
        >
          <Table size="small" aria-label="song raw tags">
            <TableBody>
              <TableRow key={`${record.id}-raw-path`}>
                <TableCell scope="row" className={classes.tableCell}>
                  {translate(`resources.song.fields.path`)}:
                </TableCell>
                <TableCell align="left">{data.path}</TableCell>
              </TableRow>
              {Object.entries(record.rawTags).map(([key, value]) => (
                <TableRow key={`${record.id}-raw-${key}`}>
                  <TableCell scope="row" className={classes.tableCell}>
                    {key}:
                  </TableCell>
                  <TableCell align="left" className={classes.value}>
                    {value.join(' • ')}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </TableContainer>
  )
}
