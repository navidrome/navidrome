import React from 'react'
import { useTranslate } from 'react-admin'
import { IconButton, Tooltip, Link } from '@material-ui/core'

import { ImLastfm2 } from 'react-icons/im'
import MusicBrainz from '../icons/MusicBrainz'
import { intersperse } from '../utils'

const ArtistExternalLinks = ({ artistInfo, record }) => {
  const translate = useTranslate()
  let links = {
    lastFM: undefined,
    musicBrainz: undefined,
  }
  let linkButtons = []
  const lastFMlink = artistInfo?.biography?.match(
    /<a\s+(?:[^>]*?\s+)?href=(["'])(.*?)\1/
  )

  if (lastFMlink) {
    links.lastFM = lastFMlink[2]
  }

  if (artistInfo && artistInfo.musicBrainzId) {
    links.musicBrainz = `https://musicbrainz.org/artist/${artistInfo.musicBrainzId}`
  }

  const addLink = (url, title, icon) => {
    if (!url) {
      return
    }
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

  addLink(links.lastFM, 'message.openIn.lastfm', <ImLastfm2 />)
  addLink(links.musicBrainz, 'message.openIn.musicbrainz', <MusicBrainz />)

  return <div>{intersperse(linkButtons, ' ')}</div>
}

export default ArtistExternalLinks
