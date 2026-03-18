import subsonic from '../subsonic'

export async function songFromRadio(radio) {
  if (!radio) {
    return undefined
  }

  let cover = subsonic.getCoverArtUrl(radio, 300)

  // If no uploaded image, try favicon as fallback
  if (!radio.uploadedImage) {
    try {
      const url = new URL(radio.homePageUrl ?? radio.streamUrl)
      url.pathname = '/favicon.ico'
      await resourceExists(url)
      cover = url.toString()
    } catch {
      // Use artwork URL (will show placeholder)
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
