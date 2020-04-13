import React from 'react'
import {
  Datagrid,
  Filter,
  FunctionField,
  List,
  NumberField,
  SearchInput,
  TextField,
  useTranslate
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import {
  DurationField,
  Pagination,
  PlayButton,
  SimpleList,
  Title
} from '../common'
import { useDispatch } from 'react-redux'
import { addTrack, setTrack } from '../audioplayer'
import AddIcon from '@material-ui/icons/Add'
import { SongBulkActions } from './SongBulkActions'
import { AlbumLinkField } from './AlbumLinkField'
import { SongDetails } from '../common'

const SongFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="title" alwaysOn />
  </Filter>
)

const SongList = (props) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  return (
    <List
      {...props}
      title={<Title subTitle={'Songs'} />}
      sort={{ field: 'title', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={<SongBulkActions />}
      filters={<SongFilter />}
      perPage={isXsmall ? 50 : 15}
      pagination={<Pagination />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(r) => (
            <>
              <PlayButton action={setTrack(r)} />
              <PlayButton action={addTrack(r)} icon={<AddIcon />} />
              {r.title}
            </>
          )}
          secondaryText={(r) => r.artist}
          tertiaryText={(r) => <DurationField record={r} source={'duration'} />}
          linkType={(id, basePath, record) => dispatch(setTrack(record))}
        />
      ) : (
        <Datagrid
          expand={<SongDetails />}
          rowClick={(id, basePath, record) => dispatch(setTrack(record))}
        >
          <TextField source="title" />
          {isDesktop && (
            <AlbumLinkField
              source="albumId"
              label={translate('resources.song.fields.album')}
              sortBy="album"
            />
          )}
          <TextField source="artist" />
          {isDesktop && <NumberField source="trackNumber" />}
          {isDesktop && <NumberField source="playCount" />}
          {isDesktop && (
            <FunctionField source="year" render={(r) => r.year || ''} />
          )}
          <DurationField source="duration" />
        </Datagrid>
      )}
    </List>
  )
}

export default SongList
