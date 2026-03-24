import subsonic from '../subsonic'
import { COVER_ART_SIZE, RADIO_PLACEHOLDER_IMAGE } from '../consts'

export async function songFromRadio(radio) {
  if (!radio) {
    return undefined
  }

  let cover = RADIO_PLACEHOLDER_IMAGE
  if (radio.uploadedImage) {
    cover = subsonic.getCoverArtUrl(radio, COVER_ART_SIZE, true)
  } else {
    // Try favicon as fallback
    try {
      const url = new URL(radio.homePageUrl ?? radio.streamUrl)
      url.pathname = '/favicon.ico'
      await resourceExists(url)
      cover = url.toString()
    } catch {
      // No cover available
    }
  }

  return {
    ...radio,
    title: radio.name,
    album: radio.homePageUrl || radio.name,
    artist: radio.name,
    cover,
    isRadio: true,
  }
}

const resourceExists = (url) => {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = function () {
      resolve(url)
    }
    img.onerror = function () {
      reject('not found')
    }
    img.src = url
  })
}
