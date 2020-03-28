import React from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  Filter,
  List,
  NumberField,
  SearchInput,
  Show,
  SimpleShowLayout,
  TextField
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import {
  BitrateField,
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

const SongFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="title" alwaysOn />
  </Filter>
)

const SongDetails = (props) => {
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField source="path" />
        <TextField source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <BitrateField source="bitRate" />
        <DateField source="updatedAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

const SongList = (props) => {
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
          {isDesktop && <TextField source="album" />}
          <TextField source="artist" />
          {isDesktop && <NumberField source="trackNumber" />}
          {isDesktop && <TextField source="maxYear" />}
          <DurationField source="duration" />
        </Datagrid>
      )}
    </List>
  )
}

export default SongList
