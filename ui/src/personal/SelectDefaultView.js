import { SelectInput, useTranslate } from 'react-admin'
import albumLists, { defaultAlbumList } from '../album/albumLists'

export const SelectDefaultView = (props) => {
  const translate = useTranslate()
  const current = localStorage.getItem('defaultView') || defaultAlbumList
  const choices = Object.keys(albumLists).map((type) => ({
    id: type,
    name: translate(`resources.album.lists.${type}`),
  }))

  return (
    <SelectInput
      {...props}
      source="defaultView"
      label={translate('menu.personal.options.defaultView')}
      defaultValue={current}
      choices={choices}
      translateChoice={false}
      onChange={(event) => {
        localStorage.setItem('defaultView', event.target.value)
      }}
    />
  )
}
