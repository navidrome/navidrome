import { SelectInput, useTranslate } from 'react-admin'
import {
  pageSizeMultipliers,
  defaultPageSizeMultiplier,
} from '../utils/pageSizes'
export const SelectPageSize = (props) => {
  const translate = useTranslate()
  const current =
    localStorage.getItem('pageSizeMultiplier') || defaultPageSizeMultiplier
  const choices = pageSizeMultipliers.map((v) => ({
    id: v,
    name: `× ${v}`,
  }))

  return (
    <SelectInput
      {...props}
      source="pageSize"
      label={translate('menu.personal.options.pageSizeMultiplier')}
      defaultValue={current}
      choices={choices}
      translateChoice={false}
      onChange={(event) => {
        localStorage.setItem('pageSizeMultiplier', event.target.value)
      }}
    />
  )
}
