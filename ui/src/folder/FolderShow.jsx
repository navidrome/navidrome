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
  useListContext,
  NumberField,
  Filter,
  SearchInput,
  ListToolbar,
} from 'react-admin'
import { useSelector } from 'react-redux'
import FolderIcon from '@material-ui/icons/Folder'
import { makeStyles, Typography, Box } from '@material-ui/core'
import Breadcrumbs from './Breadcrumbs'
import FolderSongs from './FolderSongs'
import FolderListActions from './FolderListActions'
import FolderGridView from './FolderGridView'
import {
  useResourceRefresh,
  Title,
  FolderContextMenu,
  DurationField,
  SizeField,
} from '../common'

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
  toolbar: {
    justifyContent: 'flex-start',
  },
})

const FolderFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const SubfoldersSection = (props) => {
  const { total, loaded, loading, filterValues } = useListContext()
  const classes = useStyles()
  const translate = useTranslate()
  const folderView = useSelector((state) => state.folderView)

  if (loaded && total === 0 && !filterValues.name) return null

  return (
    <>
      <Box className={classes.sectionTitle}>
        <Typography variant="h6">
          {translate('resources.folder.fields.subfolders')}
        </Typography>
      </Box>
      <ListToolbar
        classes={{ toolbar: classes.toolbar }}
        filters={<FolderFilter />}
        actions={null}
        {...props}
      />
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
    </>
  )
}

const SongsSection = ({ record, ...props }) => {
  const { total, loaded, filterValues } = useListContext()
  const classes = useStyles()
  const translate = useTranslate()

  if (loaded && total === 0 && !filterValues.title) return null

  return (
    <>
      <Box className={classes.sectionTitle}>
        <Typography variant="h6">
          {translate('resources.folder.fields.songs')}
        </Typography>
      </Box>
      <FolderSongs folder={record} {...props} />
    </>
  )
}

const FolderShowLayout = (props) => {
  const { record, loading } = useShowContext(props)
  useResourceRefresh('folder', 'song')

  if (loading || !record) return null

  return (
    <>
      <RaTitle title={<Title subTitle={record.name} />} />
      <FolderListActions {...props} />
      <SimpleShowLayout>
        <FolderHeader />

        <ReferenceManyField
          reference="folder"
          target="parent_id"
          label=""
          sort={{ field: 'name', order: 'ASC' }}
          perPage={500}
          pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
          fullWidth
        >
          <SubfoldersSection {...props} />
        </ReferenceManyField>

        <ReferenceManyField
          reference="song"
          target="folder_id"
          label=""
          sort={{ field: 'path', order: 'ASC' }}
          perPage={500}
          pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
          fullWidth
        >
          <SongsSection record={record} />
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
