import React, { Fragment } from 'react'
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
  TextField,
  TextInput,
  SimpleList
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { BitrateField, DurationField, Pagination, Title } from '../common'
import AddToQueueButton from './AddToQueueButton'
import PlayButton from './PlayButton'
import { useDispatch } from 'react-redux'
import { setTrack, addTrack } from '../player'
import AddIcon from '@material-ui/icons/Add'

const SongFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="title" alwaysOn />
    <TextInput source="album" />
    <TextInput source="artist" />
  </Filter>
)

const SongBulkActionButtons = (props) => (
  <Fragment>
    <AddToQueueButton {...props} />
  </Fragment>
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
      bulkActionButtons={<SongBulkActionButtons />}
      filters={<SongFilter />}
      perPage={isXsmall ? 50 : 15}
      pagination={<Pagination />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(record) => (
            <>
              <PlayButton record={record} />
              <PlayButton
                record={record}
                action={addTrack}
                icon={<AddIcon />}
              />
              {record.title}
            </>
          )}
          secondaryText={(record) => record.artist}
          tertiaryText={(record) => (
            <DurationField record={record} source={'duration'} />
          )}
          linkType={false}
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
          {isDesktop && <TextField source="year" />}
          <DurationField source="duration" />
        </Datagrid>
      )}
    </List>
  )
}

export default SongList
