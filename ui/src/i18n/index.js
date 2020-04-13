import en from './en'
import pt from './pt'

// When adding a new translation, import it above and add it to the list bellow

const allLanguages = { en, pt }

// "Hack" to make "albumSongs" resource use the same translations as "song"
Object.keys(allLanguages).forEach(
  (k) => (allLanguages[k].resources.albumSong = allLanguages[k].resources.song)
)

export default allLanguages
