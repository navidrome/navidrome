import React, { useEffect, useState } from 'react'
import { SelectInput, useTranslate } from 'react-admin'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'

export const UserTagFilterInput = (props) => {
  const translate = useTranslate()
  const [choices, setChoices] = useState([])

  useEffect(() => {
    httpClient(`${REST_URL}/mediaFileTag/names`)
      .then((res) => {
        setChoices((res.json || []).map((name) => ({ id: name, name })))
      })
      .catch(() => {
        // No tags yet, or request failed - leave choices empty rather than break the filter panel
      })
  }, [])

  return (
    <SelectInput
      {...props}
      label={translate('resources.song.fields.userTag')}
      source="user_tag"
      choices={choices}
      emptyText="-- None --"
    />
  )
}
