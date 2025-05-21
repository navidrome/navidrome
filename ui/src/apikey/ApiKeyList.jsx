import React, { useState } from 'react'
import {
  CreateButton,
  Datagrid,
  DateField,
  Filter,
  FunctionField,
  List,
  SearchInput,
  TextField,
  useNotify,
  useTranslate,
} from 'react-admin'
import {
  IconButton,
  makeStyles,
  Tooltip,
  useMediaQuery,
} from '@material-ui/core'
import { SimpleList } from '../common'
import AddIcon from '@material-ui/icons/Add'
import VisibilityIcon from '@material-ui/icons/Visibility'
import VisibilityOffIcon from '@material-ui/icons/VisibilityOff'
import FileCopyIcon from '@material-ui/icons/FileCopy'

const useStyles = makeStyles({
  actionContainer: {
    display: 'flex',
    alignItems: 'center',
  },
  keyContainer: {
    display: 'flex',
    alignItems: 'center',
    flex: 1,
  },
  visibilityButton: {
    padding: 4,
  },
  copyButton: {
    padding: 4,
  },
})

const ApiKeyFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const MaskedKeyField = ({ record, isSimpleMode }) => {
  const [visible, setVisible] = useState(false)
  const classes = useStyles()
  const notify = useNotify()

  if (!record || !record.key) return null

  const maskKey = (key) => {
    return '*'.repeat(key.length)
  }

  const handleCopy = (e) => {
    if (isSimpleMode) {
      e.preventDefault()
    } else {
      e.stopPropagation()
    }
    navigator.clipboard.writeText(record.key)
    notify('API key copied to clipboard', 'info')
  }

  const toggleVisibility = (e) => {
    if (isSimpleMode) {
      e.preventDefault()
    } else {
      e.stopPropagation()
    }
    setVisible(!visible)
  }

  const keyVisibilityButton = (
    <IconButton
      className={classes.visibilityButton}
      onClick={toggleVisibility}
      size="small"
    >
      {visible ? (
        <VisibilityOffIcon fontSize="small" />
      ) : (
        <VisibilityIcon fontSize="small" />
      )}
    </IconButton>
  )
  const copyButton = (
    <IconButton
      className={classes.copyButton}
      onClick={handleCopy}
      size="small"
    >
      <FileCopyIcon fontSize="small" />
    </IconButton>
  )

  return (
    <div className={classes.actionContainer}>
      <div className={classes.keyContainer}>
        {visible ? record.key : maskKey(record.key)}
      </div>
      {isSimpleMode ? (
        <>{keyVisibilityButton}</>
      ) : (
        <Tooltip title={visible ? 'Hide key' : 'Show key'}>
          {keyVisibilityButton}
        </Tooltip>
      )}
      {isSimpleMode ? (
        <>{copyButton}</>
      ) : (
        <Tooltip title="Copy to clipboard">{copyButton}</Tooltip>
      )}
    </div>
  )
}

const ApiKeyList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const translate = useTranslate()
  return (
    <List
      {...props}
      actions={
        <CreateButton
          basePath="/apikey"
          icon={<AddIcon />}
          label={translate('resources.apikey.actions.add')}
        />
      }
      sort={{ field: 'createdAt', order: 'DESC' }}
      exporter={false}
      bulkActionButtons={false}
      filters={<ApiKeyFilter />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(r) => r.name}
          secondaryText={(r) => (
            <MaskedKeyField record={r} isSimpleMode={true} />
          )}
          tertiaryText={(r) => <DateField record={r} source="createdAt" />}
          linkType={'edit'}
        />
      ) : (
        <Datagrid rowClick="edit">
          <TextField source="name" />
          <FunctionField
            label="Key"
            render={(record) => (
              <MaskedKeyField isSimpleMode={false} record={record} />
            )}
          />
          <DateField source="createdAt" showTime />
        </Datagrid>
      )}
    </List>
  )
}

export default ApiKeyList
