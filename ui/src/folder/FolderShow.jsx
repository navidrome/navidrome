import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
  Title as RaTitle,
  Datagrid,
  FunctionField,
  SimpleShowLayout,
  useTranslate,
  Pagination,
} from 'react-admin'
import { useSelector } from 'react-redux'
import FolderIcon from '@material-ui/icons/Folder'
import { makeStyles, Typography, Box } from '@material-ui/core'
import Breadcrumbs from './Breadcrumbs'
import FolderSongs from './FolderSongs'
import FolderListActions from './FolderListActions'
import FolderGridView from './FolderGridView'
import { useResourceRefresh, Title, FolderContextMenu } from '../common'

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
  row: {
    '&:hover': {
      '& $contextMenu': {
        visibility: 'visible',
      },
    },
  },
  contextMenu: {
    visibility: 'hidden',
  },
})

const FolderShowLayout = (props) => {
  const { record, loading } = useShowContext(props)
  const classes = useStyles()
  const translate = useTranslate()
  const folderView = useSelector((state) => state.folderView)
  useResourceRefresh('folder', 'song')

  if (loading || !record) return null

  return (
    <>
      <RaTitle title={<Title subTitle={record.name} />} />
      <FolderListActions {...props} />
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
          perPage={500}
          pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
          fullWidth
        >
          {folderView.grid ? (
            <FolderGridView {...props} />
          ) : (
            <Datagrid rowClick="show" classes={{ row: classes.row }}>
              <FunctionField
                source="name"
                render={(record) => (
                  <>
                    <FolderIcon className={classes.icon} />
                    {record.name}
                  </>
                )}
              />
              <FolderContextMenu
                source="name"
                className={classes.contextMenu}
                showLove={false}
              />
            </Datagrid>
          )}
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
          perPage={500}
          pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
          fullWidth
        >
          <FolderSongs folder={record} />
        </ReferenceManyField>
      </SimpleShowLayout>
    </>
  )
}

const FolderShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <FolderShowLayout {...props} {...controllerProps} />
    </ShowContextProvider>
  )
}

const FolderHeader = () => {
  const { record } = useShowContext()
  if (!record || !record.breadcrumbs) return null
  return <Breadcrumbs breadcrumbs={record.breadcrumbs} />
}

export default FolderShow
