import { Button, Chip, makeStyles, useMediaQuery } from '@material-ui/core'
import PropTypes from 'prop-types'
import { cloneElement, useState } from 'react'
import {
  BooleanField,
  BooleanInput,
  CloneButton,
  Datagrid,
  EditButton,
  Filter,
  FunctionField,
  ImageField,
  List,
  NumberField,
  sanitizeListRestProps,
  SearchInput,
  SimpleList,
  TextField,
  TextInput,
  TopToolbar,
  UrlField,
  useDataProvider,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { setTrack } from '../actions'
import { ToggleFieldsMenu, useSelectedFields } from '../common'
import config from '../config'
import { songFromRadio } from '../radio/helper'
import { StreamField } from '../radio/StreamField'

const useStyles = makeStyles({
  image: {
    '& img': {
      maxHeight: '50px',
    },
  },
})

const RadioInfoFilter = (props) => {
  return (
    <Filter {...props} variant={'outlined'}>
      <SearchInput id="search" source="name" alwaysOn />
      <TextInput id="country" source="country" />
      <TextInput id="tag" source="tags" />
      <BooleanInput source="https" />
      <BooleanInput source="existing" />
      <TextInput id="id" source="id" />
    </Filter>
  )
}

const RadioInfoActions = ({
  className,
  filters,
  resource,
  showFilter,
  displayedFilters,
  filterValues,
  ...rest
}) => {
  const isNotSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {filters &&
        cloneElement(filters, {
          resource,
          showFilter,
          displayedFilters,
          filterValues,
          context: 'button',
        })}
      {isNotSmall && <ToggleFieldsMenu resource="radioInfo" />}
    </TopToolbar>
  )
}

const BASE_TAGS_TO_SHOW = 5

const TagField = (props) => {
  const translate = useTranslate()
  const [showAll, setShowAll] = useState(false)
  const record = useRecordContext(props)

  if (!record || !record.tags) {
    return <span></span>
  }

  const tags = record.tags

  const tagList = tags.split(',')
  const isLong = tagList.length > BASE_TAGS_TO_SHOW

  const tagLength = !isLong || showAll ? tagList.length : BASE_TAGS_TO_SHOW

  const children = []

  for (let idx = 0; idx < tagLength; idx += 1) {
    children.push(<Chip key={idx} label={tagList[idx]} />)
  }

  if (isLong) {
    children.push(
      <Button
        key={-1}
        onClick={(event) => {
          setShowAll(!showAll)
          event.preventDefault()
          event.stopPropagation()
        }}
      >
        {translate(`resources.radioInfo.actions.${showAll ? 'hide' : 'show'}`)}
      </Button>
    )
  }

  return <span>{children}</span>
}

TagField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

const NewRadioButton = ({ record }) => {
  const {
    bitrate,
    codec,
    favicon,
    homepage,
    id,
    name,
    tags,
    url,
    url_resolved,
  } = record

  return (
    <CloneButton
      label="ra.action.create"
      basePath="/radios"
      record={{
        bitrate,
        codec,
        favicon,
        homepageUrl: homepage,
        name,
        radioInfoId: id,
        streamUrl: url_resolved || url,
        tags,
      }}
    />
  )
}

export const RadioInfoList = (props) => {
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const classes = useStyles()

  const toggleableFields = {
    image: (
      <ImageField sortable={false} source="favicon" className={classes.image} />
    ),
    name: <TextField source="name" />,
    homepage: (
      <UrlField
        source="homepage"
        onClick={(e) => e.stopPropagation()}
        target="_blank"
        rel="noopener noreferrer"
      />
    ),
    url: <FunctionField render={(r) => r.url_resolved || r.url} />,
    tags: <TagField source="tags" />,
    country: <TextField source="countryCode" />,
    bitrate: <NumberField source="bitrate" />,
    codec: <TextField source="codec" />,
    existingId: <BooleanField source="existingId" looseValue={true} />,
  }

  const columns = useSelectedFields({
    resource: 'radioInfo',
    columns: toggleableFields,
    defaultOff: ['bitrate', 'codec', 'country', 'image', 'url'],
  })

  const handleRowClick = async (
    id,
    _basePath,
    { bitrate, codec, favicon, homepage, url, name }
  ) => {
    if (config.sendRadioClicks) {
      dataProvider.radioClick(id)
    }
    dispatch(
      setTrack(
        await songFromRadio({
          bitRate: bitrate,
          favicon,
          homePageUrl: homepage,
          infoId: id,
          name,
          streamUrl: url,
          suffix: codec,
        })
      )
    )
  }

  return (
    <List
      {...props}
      exporter={false}
      sort={{ field: 'name', order: 'ASC' }}
      bulkActionButtons={false}
      actions={<RadioInfoActions />}
      filters={<RadioInfoFilter />}
      perPage={isXsmall ? 25 : 10}
      title={'resources.radioInfo.attribution'}
    >
      {isXsmall ? (
        <SimpleList
          rightIcon={(record) => <NewRadioButton record={record} />}
          leftIcon={({ favicon, homepage, name, url }) => (
            <StreamField
              record={{
                favicon,
                homePageUrl: homepage,
                name,
                streamUrl: url,
              }}
              source={'streamUrl'}
              hideUrl
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
              }}
            />
          )}
          linkType={''}
          primaryText={(r) => r.name}
          secondaryText={(r) => r.homepage || r.url}
        />
      ) : (
        <Datagrid rowClick={handleRowClick}>
          {columns}
          <FunctionField
            render={(record) =>
              record.existingId ? (
                <EditButton
                  basePath="/radios"
                  record={{
                    id: record.existingId,
                  }}
                />
              ) : (
                <NewRadioButton record={record} />
              )
            }
          />
        </Datagrid>
      )}
    </List>
  )
}

export default RadioInfoList
