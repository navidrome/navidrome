import deepmerge from 'deepmerge'
import en from './en'
// import fr from './fr'
import it from './it'
import pt from './pt'
import cn from './cn'

const addLanguages = (lang) => {
  Object.keys(lang).forEach((l) => (languages[l] = deepmerge(en, lang[l])))
}
const languages = { en }

// Add new languages to the object bellow
addLanguages({ cn, it, pt })

// "Hack" to make "albumSongs" resource use the same translations as "song"
Object.keys(languages).forEach(
  (k) => (languages[k].resources.albumSong = languages[k].resources.song)
)

export default languages
