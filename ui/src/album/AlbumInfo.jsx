import React, { useState, useCallback, useEffect } from 'react'
import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import { humanize, underscore } from 'inflection'
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
  useNotify,
  useRefresh,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import {
  Button,
  TextField as MuiTextField,
  CircularProgress,
} from '@material-ui/core'
import EditIcon from '@material-ui/icons/Edit'
import {
  ArtistLinkField,
  MultiLineTextField,
  ParticipantsInfo,
  RangeField,
} from '../common'
import config from '../config'
import httpClient from '../dataProvider/httpClient'

const useStyles = makeStyles({
  tableCell: {
    width: '17.5%',
  },
  value: {
    whiteSpace: 'pre-line',
  },
})

const EDITABLE_FIELDS = ['name', 'albumArtist', 'genre', 'year']

const AlbumInfo = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const record = useRecordContext(props)
  const notify = useNotify()
  const refresh = useRefresh()
  const [isEditing, setIsEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    albumArtist: '',
    genre: '',
    year: '',
  })

  useEffect(() => {
    if (record && isEditing) {
      setFormData({
        name: record.name || '',
        albumArtist: record.albumArtist || '',
        genre: record.genres?.map((g) => g.name).join(' • ') || '',
        year: record.year || '',
      })
    }
  }, [record, isEditing])

  const startEdit = useCallback(() => {
    setFormData({
      name: record.name || '',
      albumArtist: record.albumArtist || '',
      genre: record.genres?.map((g) => g.name).join(' • ') || '',
      year: record.year || '',
    })
    setIsEditing(true)
  }, [record])

  const cancelEdit = useCallback(() => {
    setIsEditing(false)
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
      album: formData.name,
      albumArtist: formData.albumArtist,
      genre: formData.genre,
      year: formData.year ? parseInt(formData.year, 10) : null,
    }
    console.log('DEBUG: Sending Payload', payload)

    try {
      await httpClient(`/api/album/${record.id}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
      notify('Album updated', { type: 'success' })
      refresh()
      setFormData({
        name: payload.album,
        albumArtist: payload.albumArtist,
        genre: payload.genre,
        year: payload.year ? String(payload.year) : '',
      })
      setIsEditing(false)
    } catch (error) {
      console.error('Error updating album:', error)
      notify('Error updating album. Check console for details.', { type: 'error' })
    } finally {
      setSaving(false)
    }
  }, [record, formData, notify, refresh])

  const buildField = (key) => {
    if (isEditing) {
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
      return null
    }

    const viewFields = {
      name: formData.name || <TextField source={'name'} />,
      libraryName: <TextField source="libraryName" />,
      albumArtist: formData.albumArtist || (
        <ArtistLinkField source="albumArtist" record={record} limit={Infinity} />
      ),
      genre: formData.genre || (
        <ArrayField source={'genres'}>
          <SingleFieldList linkType={false}>
            <ChipField source={'name'} />
          </SingleFieldList>
        </ArrayField>
      ),
      date:
        record?.maxYear && record.maxYear === record.minYear ? (
          formData.year ? parseInt(formData.year, 10) : <TextField source={'date'} />
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
    return viewFields[key]
  }

  const allFields = [
    'name',
    'libraryName',
    'albumArtist',
    'genre',
    'date',
    'originalDate',
    'releaseDate',
    'recordLabel',
    'catalogNum',
    'releaseType',
    'media',
    'grouping',
    'mood',
    'compilation',
    'updatedAt',
    'comment',
  ]

  const optionalFields = ['comment', 'genre', 'catalogNum']
  const optionalTags = ['releaseType', 'recordLabel', 'grouping', 'mood', 'media']
  const editableExceptions = ['libraryName', 'date', 'originalDate', 'releaseDate', 'recordLabel', 'catalogNum', 'releaseType', 'media', 'grouping', 'mood', 'compilation', 'updatedAt', 'comment']

  let fieldsToShow = allFields.filter((field) => {
    if (!isEditing && optionalFields.includes(field) && !record[field]) return false
    if (!isEditing && optionalTags.includes(field)) {
      if (!record?.tags?.[field.toLowerCase()]) return false
    }
    if (isEditing && !EDITABLE_FIELDS.includes(field) && !editableExceptions.includes(field)) {
      return false
    }
    return true
  })

  return (
    <TableContainer>
      <div style={{ textAlign: 'right', marginBottom: 8 }}>
        {config.enableTagEditing && !isEditing && (
          <Button
            startIcon={<EditIcon />}
            onClick={startEdit}
            variant="outlined"
            size="small"
          >
            {translate('ra.action.edit')}
          </Button>
        )}
        {isEditing && (
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
      <Table aria-label="album details" size="small">
        <TableBody>
          {fieldsToShow.map((key) => {
            const cellContent = buildField(key)
            if (!cellContent) return null
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
                  {cellContent}
                </TableCell>
              </TableRow>
            )
          })}
          {!isEditing && <ParticipantsInfo record={record} classes={classes} />}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

export default AlbumInfo
