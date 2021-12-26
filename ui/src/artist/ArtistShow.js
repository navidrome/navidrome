import React, { useState, createElement, useEffect } from 'react'
import { useMediaQuery } from '@material-ui/core'
import {
  useShowController,
  ShowContextProvider,
  useRecordContext,
  useShowContext,
  ReferenceManyField,
} from 'react-admin'
import subsonic from '../subsonic'
import AlbumGridView from '../album/AlbumGridView'
import MobileArtistDetails from './MobileArtistDetails'
import DesktopArtistDetails from './DesktopArtistDetails'

const ArtistDetails = (props) => {
  const showContext = useShowContext(props)
  const record = useRecordContext(props)
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  const [artistInfo, setArtistInfo] = useState()
  const [topSong, setTopSong] = useState()

  const biography =
    artistInfo?.biography?.replace(new RegExp('<.*>', 'g'), '') ||
    record.biography
  const img = artistInfo?.largeImageUrl || record.largeImageUrl

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
        console.error('error on artist page', e)
      })

    subsonic
      .getTopSongs(record.name, record.id)
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          setTopSong(data.topSongs.song)
        }
      })
      .catch((e) => {
        console.error('error on artist page', e)
      })
  }, [record])

  const component = isDesktop ? DesktopArtistDetails : MobileArtistDetails
  return (
    <>
      {createElement(component, {
        img,
        artistInfo,
        record,
        biography,
        topSong,
        showContext,
      })}
    </>
  )
}

const AlbumShowLayout = (props) => {
  const showContext = useShowContext(props)
  const record = useRecordContext()

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
          perPage={0}
          pagination={null}
        >
          <AlbumGridView {...props} />
        </ReferenceManyField>
      )}
    </>
  )
}

const ArtistShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <AlbumShowLayout {...controllerProps} />
    </ShowContextProvider>
  )
}

export default ArtistShow
