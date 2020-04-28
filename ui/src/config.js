// These defaults are only used in development mode. When bundled in the app,
// the __APP_CONFIG__ object is dynamically filled by the ServeIndex function,
// in the /server/app/serve_index.go
const defaultConfig = {
  version: 'dev',
  firstTime: false,
  baseURL: '',
  loginBackgroundURL: 'https://source.unsplash.com/random/1600x900?music',
  enableTranscodingConfig: true,
}

let config

try {
  const appConfig = JSON.parse(window.__APP_CONFIG__)

  config = {
    ...defaultConfig,
    ...appConfig,
  }
} catch (e) {
  config = defaultConfig
}

export default config
