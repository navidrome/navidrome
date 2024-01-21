import { Link } from '@material-ui/core'
import {
  forwardRef,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  BooleanField,
  BooleanInput,
  Create,
  Datagrid,
  DateField,
  List,
  Loading,
  SaveButton,
  SelectInput,
  SimpleForm,
  TextField,
  TextInput,
  Toolbar,
  useMutation,
  useNotify,
  useRecordContext,
  useRedirect,
  useRefresh,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'

const Expand = ({ record }) => (
  <div>
    <div dangerouslySetInnerHTML={{ __html: record.description }} />
    <Link href={record.url} target="_blank" rel="noopener noreferrer">
      {record.url}
    </Link>
  </div>
)

const NameInput = (props) => {
  const { id, name } = useRecordContext(props)

  return (
    <TextInput
      multiline
      fullWidth
      name={`name[${id}]`}
      defaultValue={name}
      parse={(val) => val || ''}
      placeholder={name}
      onClick={(event) => {
        event.stopPropagation()
      }}
    />
  )
}

const SyncInput = (props) => {
  const { id } = useRecordContext(props)

  return (
    <BooleanInput
      sortable={false}
      name={`sync[${id}]`}
      label=""
      onClick={(event) => {
        event.stopPropagation()
      }}
    />
  )
}

const MyDataGrid = forwardRef(
  ({ onUnselectItems, selectedIds, setIds, ...props }, ref) => {
    useEffect(() => {
      setIds(selectedIds)
    }, [selectedIds, setIds])

    useEffect(() => {
      return () => {
        // This will run on dismount to clear up state
        onUnselectItems()
      }
    }, [onUnselectItems])

    const canSync = useMemo(() => {
      let canSync = false
      for (const id of props.ids) {
        canSync ||= props.data[id].syncable
      }
      return canSync
    }, [props.data, props.ids])

    ref.current = onUnselectItems

    return (
      <Datagrid
        {...props}
        expand={<Expand />}
        rowClick="toggleSelection"
        selectedIds={selectedIds}
      >
        <NameInput source="name" sortable={false} />
        {canSync && <SyncInput source="sync" sortable={false} />}
        <TextField source="creator" sortable={false} />
        <DateField source="createdAt" sortable={false} />
        <DateField source="updatedAt" sortable={false} />
        <BooleanField source="existing" sortable={false} />
      </Datagrid>
    )
  },
)

const Dummy = () => <span></span>

const ExternalPlaylistSelect = forwardRef(
  ({ fullWidth, playlists, setIds, filter, ...props }, ref) => {
    return (
      <>
        <List
          {...props}
          filter={filter}
          title={<span></span>}
          bulkActionButtons={<Dummy />}
          exporter={false}
          actions={<Dummy />}
        >
          <MyDataGrid setIds={setIds} ref={ref} />
        </List>
      </>
    )
  },
)

const ExternalPlaylistCreate = (props) => {
  const clearRef = useRef()

  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const refresh = useRefresh()
  const translate = useTranslate()

  const [agents, setAgents] = useState(null)
  const [selectedAgent, setSelectedAgent] = useState(null)
  const [selectedType, setSelectedType] = useState(null)
  const [ids, setIds] = useState([])

  const resourceName = translate('resources.externalPlaylist.name', {
    smart_count: 1,
  })

  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })

  useEffect(() => {
    httpClient(`${REST_URL}/externalPlaylist/agents`)
      .then((resp) => {
        const mapping = {}
        for (const agent of resp.json) {
          mapping[agent.name] = agent.types
        }
        setAgents(mapping)
      })
      .catch((err) => {
        console.log(err)
      })
  }, [])

  useEffect(() => {
    if (clearRef.current) {
      clearRef.current()
    }
  }, [])

  const allAgents = useMemo(
    () =>
      agents === null
        ? []
        : Object.keys(agents).map((k) => ({ id: k, name: k })),
    [agents],
  )

  const agentKeys = useMemo(
    () =>
      selectedAgent === null
        ? []
        : agents[selectedAgent].map((type) => ({
            id: type,
            name: translate(
              `resources.externalPlaylist.agent.${selectedAgent}.${type}`,
            ),
          })),
    [agents, selectedAgent, translate],
  )

  const changeAgent = (event) => {
    if (clearRef.current) {
      clearRef.current()
    }
    setSelectedAgent(event.target.value)
  }

  const changeType = (event) => {
    if (clearRef.current) {
      clearRef.current()
    }
    setSelectedType(event.target.value)
  }

  const save = useCallback(
    async (values) => {
      const { agent, name, type, update } = values
      const playlists = {}

      let sync = values.sync ?? {}

      let count = 0

      for (const id of ids) {
        playlists[id] = {
          name: name[id],
          sync: sync[id],
        }
        count++
      }

      try {
        await mutate(
          {
            type: 'create',
            resource: 'externalPlaylist',
            payload: {
              data: { agent, type, update, playlists },
            },
          },
          { returnPromise: true },
        )
        notify('resources.externalPlaylist.notifications.created', 'info', {
          smart_count: count,
        })
        refresh()
        redirect('/playlist')
      } catch (error) {
        notify('resources.externalPlaylist.notifications.failed', 'error', {
          cause: error.body.error,
        })
      }
    },
    [ids, mutate, notify, redirect, refresh],
  )

  let formBody

  if (allAgents.length === 0) {
    formBody = <div>{translate('message.noPlaylistAgent')}</div>
  } else {
    formBody = [
      <SelectInput source="agent" choices={allAgents} onChange={changeAgent} />,
      <SelectInput source="type" choices={agentKeys} onChange={changeType} />,
      <BooleanInput source="update" defaultValue={true} />,
      selectedType && (
        <ExternalPlaylistSelect
          filter={{ agent: selectedAgent, type: selectedType }}
          setIds={setIds}
          fullWidth
          ref={clearRef}
        />
      ),
    ]
  }

  return (
    <Create title={<Title subTitle={title} />} {...props}>
      {agents === null ? (
        <Loading />
      ) : (
        <SimpleForm
          toolbar={
            <Toolbar>
              <SaveButton disabled={ids.length === 0} />
            </Toolbar>
          }
          save={save}
        >
          {formBody}
        </SimpleForm>
      )}
    </Create>
  )
}

export default ExternalPlaylistCreate
