import config from '../config'
import subsonic from '../subsonic'

export async function songFromRadio(radio) {
  if (!radio) {
    return undefined
  }

  let cover = 'internet-radio-icon.svg'
  try {
    const url = new URL(radio.homePageUrl ?? radio.streamUrl)
    url.pathname = '/favicon.ico'

    let urlString

    if (config.enableProxy) {
      urlString = subsonic.url('proxy/icon', '', {
        url: url.toString(),
      })
    } else {
      urlString = url.toString()
    }

    await resourceExists(urlString)
    cover = urlString
  } catch {}

  let streamUrl

  if (config.enableProxy) {
    streamUrl = subsonic.url('proxy/stream', '', {
      url: radio.streamUrl,
    })
  } else {
    streamUrl = radio.streamUrl
  }

  return { ...radio, cover, streamUrl }
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
