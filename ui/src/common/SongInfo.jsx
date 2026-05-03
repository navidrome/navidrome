import React, { useState, useCallback, useEffect } from 'react'
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
  useNotify,
  useRefresh,
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
import {
  Button,
  TextField as MuiTextField,
  CircularProgress,
} from '@material-ui/core'
import EditIcon from '@material-ui/icons/Edit'
import config from '../config'
import { AlbumLinkField } from '../song/AlbumLinkField'
import { Tab, Tabs } from '@material-ui/core'
import httpClient from '../dataProvider/httpClient'

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

const EDITABLE_FIELDS = [
  'title',
  'artist',
  'albumArtist',
  'album',
  'genre',
  'year',
  'trackNumber',
]

const READONLY_FIELDS = [
  'path',
  'libraryName',
  'discSubtitle',
  'bitRate',
  'bitDepth',
  'sampleRate',
  'channels',
  'size',
  'updatedAt',
  'playCount',
  'bpm',
  'comment',
  'compilation',
  'playDate',
  'albumGain',
  'trackGain',
]

export const SongInfo = (props) => {
  const classes = useStyles({ gain: config.enableReplayGain })
  const translate = useTranslate()
  const record = useRecordContext(props)
  const notify = useNotify()
  const refresh = useRefresh()
  const [tab, setTab] = useState(0)
  const [editMode, setEditMode] = useState(false)
  const [saving, setSaving] = useState(false)
  const [formData, setFormData] = useState({
    title: '',
    artist: '',
    albumArtist: '',
    album: '',
    genre: '',
    year: '',
    trackNumber: '',
  })

  useEffect(() => {
    if (record && editMode) {
      setFormData({
        title: record.title || '',
        artist: record.artist || '',
        albumArtist: record.albumArtist || '',
        album: record.album || '',
        genre: record.genres?.map((g) => g.name).join(' • ') || '',
        year: record.year || '',
        trackNumber: record.trackNumber || '',
      })
    }
  }, [record, editMode])

  const startEdit = useCallback(() => {
    setFormData({
      title: record.title || '',
      artist: record.artist || '',
      albumArtist: record.albumArtist || '',
      album: record.album || '',
      genre: record.genres?.map((g) => g.name).join(' • ') || '',
      year: record.year || '',
      trackNumber: record.trackNumber || '',
    })
    setEditMode(true)
  }, [record])

  const cancelEdit = useCallback(() => {
    setEditMode(false)
  }, [])

  const handleFieldChange = useCallback((field) => (event) => {
    setFormData((prev) => ({
      ...prev,
      [field]: event.target.value,
    }))
  }, [])

  const handleSave = useCallback(async () => {
    if (!record?.id) return

    setSaving(true)
    const payload = {
      title: formData.title,
      artist: formData.artist,
      album: formData.album,
      albumArtist: formData.albumArtist,
      genre: formData.genre,
      year: formData.year ? parseInt(formData.year, 10) : null,
      trackNumber: formData.trackNumber ? parseInt(formData.trackNumber, 10) : null,
    }

    try {
      const response = await httpClient(`/api/song/${record.id}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
      console.log('Song update response:', response)
      notify('Song updated successfully', { type: 'success' })
      refresh()
      setEditMode(false)
      setFormData({
        title: payload.title,
        artist: payload.artist,
        album: payload.album,
        albumArtist: payload.albumArtist,
        genre: payload.genre,
        year: payload.year ? String(payload.year) : '',
        trackNumber: payload.trackNumber ? String(payload.trackNumber) : '',
      })
    } catch (error) {
      console.error('Error updating song:', error)
      notify('Error updating song. Check console for details.', { type: 'error' })
    } finally {
      setSaving(false)
    }
  }, [record, formData, notify, refresh])

  const excludedTags = [
    'genre',
    'disctotal',
    'tracktotal',
    'releasetype',
    'recordlabel',
    'media',
    'albumversion',
  ]

  const buildRow = (key) => {
    if (editMode) {
      if (EDITABLE_FIELDS.includes(key)) {
        return (
          <MuiTextField
            value={formData[key] || ''}
            onChange={handleFieldChange(key)}
            variant="outlined"
            size="small"
            fullWidth
            disabled={saving}
          />
        )
      }
      if (READONLY_FIELDS.includes(key)) {
        const readOnlyFields = {
          path: <PathField />,
          libraryName: <TextField source="libraryName" />,
          discSubtitle: <TextField source="discSubtitle" />,
          bitRate: <BitrateField source="bitRate" />,
          bitDepth: <NumberField source="bitDepth" />,
          sampleRate: <NumberField source="sampleRate" />,
          channels: <NumberField source="channels" />,
          size: <SizeField source="size" />,
          updatedAt: <DateField source="updatedAt" showTime />,
          playCount: <TextField source="playCount" />,
          bpm: <NumberField source="bpm" />,
          comment: <MultiLineTextField source="comment" />,
          compilation: <BooleanField source="compilation" />,
        }
        return readOnlyFields[key] || null
      }
      return null
    }

    const viewFields = {
      title: formData.title || <TextField source="title" />,
      libraryName: <TextField source="libraryName" />,
      album: formData.album || <AlbumLinkField source="album" sortByOrder={'ASC'} record={record} />,
      discSubtitle: <TextField source="discSubtitle" />,
      albumArtist: formData.albumArtist || <ArtistLinkField source="albumArtist" record={record} limit={Infinity} />,
      artist: formData.artist || <ArtistLinkField source="artist" record={record} limit={Infinity} />,
      genre: formData.genre || <FunctionField render={(r) => r.genres?.map((g) => g.name).join(' • ')} />,
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
      year: formData.year ? parseInt(formData.year, 10) : <NumberField source="year" />,
      trackNumber: formData.trackNumber ? parseInt(formData.trackNumber, 10) : <NumberField source="trackNumber" />,
    }
    return viewFields[key] || null
  }

  const allFields = [
    'title',
    'artist',
    'albumArtist',
    'album',
    'genre',
    'year',
    'trackNumber',
    'path',
    'libraryName',
    'discSubtitle',
    'bitRate',
    'bitDepth',
    'sampleRate',
    'channels',
    'size',
    'updatedAt',
    'playCount',
    'bpm',
    'comment',
    'compilation',
  ]

  const optionalFields = [
    'discSubtitle',
    'comment',
    'bpm',
    'genre',
    'bitDepth',
    'sampleRate',
  ]
  const fieldsToShow = allFields.filter((field) => {
    if (editMode) return true
    if (!record[field] && optionalFields.includes(field.toLowerCase())) return false
    if (field === 'playCount' && record.playCount <= 0) return false
    return true
  })

  if (editMode && record.playCount > 0) {
    if (!fieldsToShow.includes('playDate')) {
      fieldsToShow.push('playDate')
    }
  }

  if (config.enableReplayGain && !editMode) {
    if (!fieldsToShow.includes('albumGain')) {
      fieldsToShow.push('albumGain')
    }
    if (!fieldsToShow.includes('trackGain')) {
      fieldsToShow.push('trackGain')
    }
  }

  const tags = Object.entries(record.tags ?? {}).filter(
    (tag) => !excludedTags.includes(tag[0]),
  )

  const showEditButton = config.enableTagEditing && !editMode
  const showSaveCancel = editMode
  const showTabs = record.rawTags && !editMode

  return (
    <TableContainer>
      {showTabs && (
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
      <div style={{ textAlign: 'right', marginBottom: 8 }}>
        {showEditButton && (
          <Button
            startIcon={<EditIcon />}
            onClick={startEdit}
            variant="outlined"
            size="small"
          >
            {translate('ra.action.edit')}
          </Button>
        )}
        {showSaveCancel && (
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <Button
              onClick={cancelEdit}
              disabled={saving}
              variant="outlined"
              size="small"
            >
              {translate('ra.action.cancel')}
            </Button>
            <Button
              onClick={handleSave}
              disabled={saving}
              variant="contained"
              color="primary"
              size="small"
              startIcon={saving ? <CircularProgress size={16} color="inherit" /> : null}
            >
              {translate('ra.action.save')}
            </Button>
          </div>
        )}
      </div>
      {showTabs ? (
        <>
          <div
            hidden={tab !== 0}
            id="mapped-tags-body"
            aria-labelledby="mapped-tags-tab"
          >
            <Table aria-label="song details" size="small">
              <TableBody>
                {fieldsToShow.map((key) => {
                  const cellContent = buildRow(key)
                  if (!cellContent) return null
                  return (
                    <TableRow key={`${record?.id}-${key}`}>
                      <TableCell scope="row" className={classes.tableCell}>
                        {translate(`resources.song.fields.${key}`, {
                          _: humanize(underscore(key)),
                        })}
                        :
                      </TableCell>
                      <TableCell align="left" className={classes.value}>
                        {cellContent}
                      </TableCell>
                    </TableRow>
                  )
                })}
                {!editMode && <ParticipantsInfo classes={classes} record={record} />}
                {tags.length > 0 && !editMode && (
                  <TableRow key={`${record?.id}-separator`}>
                    <TableCell scope="row" className={classes.tableCell} />
                    <TableCell align="left" className={classes.value}>
                      <h4>{translate(`resources.song.fields.tags`)}</h4>
                    </TableCell>
                  </TableRow>
                )}
                {tags.map(([name, values]) => (
                  <TableRow key={`${record?.id}-tag-${name}`}>
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
          <div
            hidden={tab !== 1}
            id="raw-tags-body"
            aria-labelledby="raw-tags-tab"
          >
            <Table size="small" aria-label="song raw tags">
              <TableBody>
                <TableRow key={`${record?.id}-raw-path`}>
                  <TableCell scope="row" className={classes.tableCell}>
                    {translate(`resources.song.fields.path`)}:
                  </TableCell>
                  <TableCell align="left">
                    <PathField />
                  </TableCell>
                </TableRow>
                {Object.entries(record.rawTags || {}).map(([key, value]) => (
                  <TableRow key={`${record?.id}-raw-${key}`}>
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
        </>
      ) : (
        <Table aria-label="song details" size="small">
          <TableBody>
            {fieldsToShow.map((key) => {
              const cellContent = buildRow(key)
              if (!cellContent) return null
              return (
                <TableRow key={`${record?.id}-${key}`}>
                  <TableCell scope="row" className={classes.tableCell}>
                    {translate(`resources.song.fields.${key}`, {
                      _: humanize(underscore(key)),
                    })}
                    :
                  </TableCell>
                  <TableCell align="left" className={classes.value}>
                    {cellContent}
                  </TableCell>
                </TableRow>
              )
            })}
            {!editMode && <ParticipantsInfo classes={classes} record={record} />}
            {tags.length > 0 && !editMode && (
              <TableRow key={`${record?.id}-separator`}>
                <TableCell scope="row" className={classes.tableCell} />
                <TableCell align="left" className={classes.value}>
                  <h4>{translate(`resources.song.fields.tags`)}</h4>
                </TableCell>
              </TableRow>
            )}
            {tags.map(([name, values]) => (
              <TableRow key={`${record?.id}-tag-${name}`}>
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
      )}
    </TableContainer>
  )
}