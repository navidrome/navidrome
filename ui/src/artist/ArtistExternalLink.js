import React from 'react'
import { useTranslate } from 'react-admin'
import { IconButton, Tooltip, Link } from '@material-ui/core'

import { ImLastfm2 } from 'react-icons/im'
import MusicBrainz from '../icons/MusicBrainz'
import { intersperse } from '../utils'
import config from '../config'

const ArtistExternalLinks = ({ artistInfo, record }) => {
  const translate = useTranslate()
  let linkButtons = []
  const lastFMlink = artistInfo?.biography?.match(
    /<a\s+(?:[^>]*?\s+)?href=(["'])(.*?)\1/
  )

  const addLink = (url, title, icon) => {
    const translatedTitle = translate(title)
    const link = (
      <Link href={url} target="_blank" rel="noopener noreferrer">
        <Tooltip title={translatedTitle}>
          <IconButton size={'small'} aria-label={translatedTitle}>
            {icon}
          </IconButton>
        </Tooltip>
      </Link>
    )
    const id = linkButtons.length
    linkButtons.push(<span key={`link-${record.id}-${id}`}>{link}</span>)
  }

  if (config.lastFMEnabled) {
    if (lastFMlink) {
      addLink(
        lastFMlink[2],
        'message.openIn.lastfm',
        <ImLastfm2 className="lastfm-icon" />
      )
    } else if (artistInfo?.lastFmUrl) {
      addLink(
        artistInfo?.lastFmUrl,
        'message.openIn.lastfm',
        <ImLastfm2 className="lastfm-icon" />
      )
    }
  }

  artistInfo?.musicBrainzId &&
    addLink(
      `https://musicbrainz.org/artist/${artistInfo.musicBrainzId}`,
      'message.openIn.musicbrainz',
      <MusicBrainz className="musicbrainz-icon" />
    )

  return <div>{intersperse(linkButtons, ' ')}</div>
}

export default ArtistExternalLinks
