export async function songFromRadio(radio) {
  if (!radio) {
    return undefined
  }

  let cover = 'internet-radio-icon.svg'
  try {
    const url = new URL(radio.homePageUrl ?? radio.streamUrl)
    url.pathname = '/favicon.ico'
    await resourceExists(url)
    cover = url.toString()
  } catch {
    // ignore
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
