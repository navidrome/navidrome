import { useState, useCallback, createContext, useContext } from 'react'

const SongEditorContext = createContext(null)

export const useSongEditor = () => {
  const context = useContext(SongEditorContext)
  if (!context) {
    throw new Error('useSongEditor must be used within SongEditorProvider')
  }
  return context
}

export const SongEditorProvider = ({ children }) => {
  const [songId, setSongId] = useState(null)
  const [song, setSong] = useState(null)

  const openEditor = useCallback((idOrSong) => {
    if (typeof idOrSong === 'object') {
      setSong(idOrSong)
      setSongId(idOrSong.id)
    } else {
      setSongId(idOrSong)
      setSong(null)
    }
  }, [])

  const closeEditor = useCallback(() => {
    setSongId(null)
    setSong(null)
  }, [])

  return (
    <SongEditorContext.Provider value={{ songId, song, openEditor, closeEditor }}>
      {children}
    </SongEditorContext.Provider>
  )
}