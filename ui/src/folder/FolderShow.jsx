import React from 'react'
import {
  Show,
  SimpleShowLayout,
  ReferenceManyField,
  Datagrid,
  useRecordContext,
  useTranslate,
  FunctionField,
} from 'react-admin'
import {
  SongDatagrid,
  SongTitleField,
  ArtistLinkField,
  DurationField,
  SongContextMenu,
} from '../common'
import FolderIcon from '@material-ui/icons/Folder'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { makeStyles, Typography, Box } from '@material-ui/core'
import Breadcrumbs from './Breadcrumbs'
import FolderActions from './FolderActions'
import config from '../config'

const useStyles = makeStyles({
  icon: {
    verticalAlign: 'middle',
    marginRight: '10px',
  },
  sectionTitle: {
    marginTop: '20px',
    marginBottom: '10px',
    fontWeight: 'bold',
  },
  contextHeader: {
    marginLeft: '3px',
    marginTop: '-2px',
    verticalAlign: 'text-top',
  },
})

const FolderShow = (props) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <Show {...props} actions={null} title={<FolderTitle />}>
      <SimpleShowLayout>
        <FolderHeader />
        <Box className={classes.sectionTitle}>
          <Typography variant="h6">
            {translate('resources.folder.fields.subfolders')}
          </Typography>
        </Box>
        <ReferenceManyField
          reference="folder"
          target="parent_id"
          label=""
          sort={{ field: 'name', order: 'ASC' }}
          fullWidth
        >
          <Datagrid rowClick="show">
            <FunctionField
              source="name"
              render={(record) => (
                <>
                  <FolderIcon className={classes.icon} />
                  {record.name}
                </>
              )}
            />
          </Datagrid>
        </ReferenceManyField>

        <Box className={classes.sectionTitle}>
          <Typography variant="h6">
            {translate('resources.folder.fields.songs')}
          </Typography>
        </Box>
        <ReferenceManyField
          reference="song"
          target="folder_id"
          label=""
          sort={{ field: 'path', order: 'ASC' }}
          fullWidth
        >
          <SongDatagrid resource="song">
            <SongTitleField source="title" showTrackNumbers={false} />
            <ArtistLinkField source="artist" sortable={false} />
            <DurationField source="duration" sortable={false} />
            <SongContextMenu
              source={'starred_at'}
              sortable={false}
              label={
                config.enableFavourites && (
                  <FavoriteBorderIcon
                    fontSize={'small'}
                    className={classes.contextHeader}
                  />
                )
              }
            />
          </SongDatagrid>
        </ReferenceManyField>
      </SimpleShowLayout>
    </Show>
  )
}

const FolderTitle = () => {
  const record = useRecordContext()
  return record && record.name ? <span>{record.name}</span> : null
}

const FolderHeader = () => {
  const record = useRecordContext()
  if (!record || !record.breadcrumbs) return null
  return <Breadcrumbs breadcrumbs={record.breadcrumbs} />
}

export default FolderShow
