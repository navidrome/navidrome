import { useMemo } from 'react'
import en from '../i18n/en.json'
import { useTranslate } from 'react-admin'

export const useArtistRoles = (plural) => {
  const translate = useTranslate()
  const count = plural ? 2 : 1

  const roles = useMemo(() => {
    const rolesObj = en?.resources?.artist?.roles

    const roles = Object.keys(rolesObj).reduce((acc, role) => {
      acc.push({
        id: role,
        name: translate(`resources.artist.roles.${role}`, {
          smart_count: count,
        }),
      })
      return acc
    }, [])
    roles?.sort((a, b) => a.name.localeCompare(b.name))

    return roles
  }, [count, translate])

  return roles
}
