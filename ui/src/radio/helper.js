export async function songFromRadio(radio) {
  if (!radio) {
    return undefined
  }

  let cover = 'internet-radio-icon.svg'
  try {
    let url

    if (radio.favicon) {
      url = new URL(radio.favicon)
    } else {
      url = new URL(radio.homePageUrl ?? radio.streamUrl)
      url.pathname = '/favicon.ico'
    }

    await resourceExists(url)
    cover = url.toString()
  } catch {}

  if (radio.codec) {
    radio.suffix = radio.codec
  }

  if (radio.bitrate) {
    radio.bitRate = radio.bitrate
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
