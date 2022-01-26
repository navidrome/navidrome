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
        console.error('error on artist page', e)
      })
  }, [record])

  if (artistInfo) {
    // Assign placeholders if image is missing
    artistInfo.largeImageUrl =
      artistInfo.largeImageUrl ||
      'https://lastfm.freetls.fastly.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png'
    artistInfo.mediumImageUrl =
      artistInfo.mediumImageUrl ||
      'https://lastfm.freetls.fastly.net/i/u/174s/2a96cbd8b46e442fc41c2b86b821562f.png'
    artistInfo.smallImageUrl =
      artistInfo.smallImageUrl ||
      'https://lastfm.freetls.fastly.net/i/u/64s/2a96cbd8b46e442fc41c2b86b821562f.png'
  }

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
