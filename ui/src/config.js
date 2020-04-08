const defaultConfig = {
  version: 'dev',
  firstTime: false,
  baseURL: '',
  loginBackgroundURL: 'https://source.unsplash.com/random/1600x900?music'
}

let config

try {
  const appConfig = JSON.parse(window.__APP_CONFIG__)

  config = {
    ...defaultConfig,
    ...appConfig
  }
} catch (e) {
  config = defaultConfig
}

export default config
