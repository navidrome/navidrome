import React, { useState, createElement, useEffect } from 'react'
import { useMediaQuery, withWidth } from '@material-ui/core'
import {
  useShowController,
  ShowContextProvider,
  useRecordContext,
  useShowContext,
  ReferenceManyField,
  Pagination,
} from 'react-admin'
import subsonic from '../subsonic'
import AlbumGridView from '../album/AlbumGridView'
import MobileArtistDetails from './MobileArtistDetails'
import DesktopArtistDetails from './DesktopArtistDetails'
import { useAlbumsPerPage } from '../common/index.js'

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
  const showContext = useShowContext(props)
  const record = useRecordContext()
  const { width } = props
  const [, perPageOptions] = useAlbumsPerPage(width)

  const maxPerPage = 90
  let perPage = 0
  let pagination = null

  const count = Math.max(
    record?.stats?.['albumartist']?.albumCount || 0,
    record?.stats?.['artist']?.albumCount ?? 0,
  )

  if (count > maxPerPage) {
    perPage = Math.trunc(maxPerPage / perPageOptions[0]) * perPageOptions[0]
    const rowsPerPageOptions = [1, 2, 3].map((option) =>
      Math.trunc(option * (perPage / 3)),
    )
    pagination = <Pagination rowsPerPageOptions={rowsPerPageOptions} />
  }

  return (
    <>
      {record && <ArtistDetails />}
      {record && (
        <ReferenceManyField
          {...showContext}
          addLabel={false}
          reference="album"
          target="artist_id"
          sort={{ field: 'max_year', order: 'ASC' }}
          filter={{ artist_id: record?.id }}
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
