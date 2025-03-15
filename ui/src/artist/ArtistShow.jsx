import React, { useState, createElement, useEffect } from 'react'
import {
  makeStyles,
  MenuItem,
  Select,
  useMediaQuery,
  withWidth,
} from '@material-ui/core'
import {
  useShowController,
  ShowContextProvider,
  useRecordContext,
  useShowContext,
  ReferenceManyField,
  Pagination,
  useTranslate,
} from 'react-admin'
import subsonic from '../subsonic'
import AlbumGridView from '../album/AlbumGridView'
import MobileArtistDetails from './MobileArtistDetails'
import DesktopArtistDetails from './DesktopArtistDetails'
import { useAlbumsPerPage } from '../common/index.js'
import { useArtistRoles } from '../common/useArtistRoles.jsx'
import { useLocation } from 'react-router-dom/cjs/react-router-dom.min.js'

const useStyles = makeStyles({
  root: {
    padding: '1em',
  },
})

const ArtistDetails = (props) => {
  const record = useRecordContext(props)
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  const [artistInfo, setArtistInfo] = useState()

  const biography =
    artistInfo?.biography?.replace(new RegExp('<.*>', 'g'), '') ||
    record.biography

  useEffect(() => {
    subsonic
      .getArtistInfo(record.id)
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          setArtistInfo(data.artistInfo)
        }
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.error('error on artist page', e)
      })
  }, [record.id])

  const component = isDesktop ? DesktopArtistDetails : MobileArtistDetails
  return (
    <>
      {createElement(component, {
        artistInfo,
        record,
        biography,
      })}
    </>
  )
}

const AlbumShowLayout = (props) => {
  const translate = useTranslate()
  const showContext = useShowContext(props)
  const record = useRecordContext()
  const { width } = props
  const [, perPageOptions] = useAlbumsPerPage(width)
  const { search } = useLocation()
  const [role, setRole] = useState(
    new URLSearchParams(search).get('role') ?? 'total',
  )
  const roles = useArtistRoles(false)
  const classes = useStyles()

  const maxPerPage = 36
  let perPage = 0
  let pagination = null

  const count =
    role === 'total'
      ? record?.albumCount
      : record?.stats?.[role]?.albumCount || 0

  if (count > maxPerPage) {
    perPage = Math.trunc(maxPerPage / perPageOptions[0]) * perPageOptions[0]
    const rowsPerPageOptions = [1, 2, 3].map((option) =>
      Math.trunc(option * perPage),
    )
    pagination = <Pagination rowsPerPageOptions={rowsPerPageOptions} />
  }

  const id = `${role}_id`

  return (
    <>
      {record && <ArtistDetails />}
      <div className={classes.root}>
        <Select
          value={role}
          onChange={(event) => setRole(event.target.value)}
          fullWidth
        >
          <MenuItem key="total" value="total">
            {translate('resources.artist.fields.allRoles')} (
            {record?.albumCount || 0})
          </MenuItem>
          {roles
            .filter((role) => record?.stats?.[role.id]?.albumCount)
            .map((role) => (
              <MenuItem key={role.id} value={role.id}>
                {role.name} ({record.stats[role.id].albumCount})
              </MenuItem>
            ))}
        </Select>
      </div>
      {record && (
        <ReferenceManyField
          {...showContext}
          addLabel={false}
          reference="album"
          target={id}
          sort={{ field: 'max_year', order: 'ASC' }}
          filter={{ [id]: record?.id }}
          perPage={perPage}
          pagination={pagination}
        >
          <AlbumGridView {...props} />
        </ReferenceManyField>
      )}
    </>
  )
}

const ArtistShow = withWidth()((props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <AlbumShowLayout {...controllerProps} />
    </ShowContextProvider>
  )
})

export default ArtistShow
