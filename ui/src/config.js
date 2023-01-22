// These defaults are only used in development mode. When bundled in the app,
// the __APP_CONFIG__ object is dynamically filled by the ServeIndex function,
// in the /server/app/serve_index.go
const defaultConfig = {
  version: 'dev',
  firstTime: false,
  baseURL: '',
  variousArtistsId: '03b645ef2100dfc42fa9785ea3102295', // See consts.VariousArtistsID in consts.go
  // Login backgrounds from https://unsplash.com/collections/1065384/music-wallpapers
  loginBackgroundURL: 'https://source.unsplash.com/collection/1065384/1600x900',
  enableTranscodingConfig: true,
  enableDownloads: true,
  enableFavourites: true,
  losslessFormats: 'FLAC,WAV,ALAC,DSF',
  welcomeMessage: '',
  gaTrackingId: '',
  devActivityPanel: true,
  enableStarRating: true,
  defaultTheme: 'Dark',
  defaultLanguage: '',
  defaultUIVolume: 100,
  enableUserEditing: true,
  devEnableShare: true,
  devSidebarPlaylists: true,
  lastFMEnabled: true,
  lastFMApiKey: '9b94a5515ea66b2da3ec03c12300327e',
  listenBrainzEnabled: true,
  enableCoverAnimation: true,
  devShowArtistPage: true,
  enableReplayGain: true,
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
