import React, { useCallback, useMemo, useState } from 'react'
import config from '../config'
import { BITRATE_CHOICES, DEFAULT_SHARE_BITRATE } from '../consts'
import {
  BooleanInput,
  SelectInput,
  useGetList,
  useTranslate,
} from 'react-admin'

export const useTranscodingOptions = () => {
  const translate = useTranslate()
  const [format, setFormat] = useState(config.defaultDownsamplingFormat)
  const [maxBitRate, setMaxBitRate] = useState(DEFAULT_SHARE_BITRATE)
  const [originalFormat, setUseOriginalFormat] = useState(true)

  const { data: formats, loading: loadingFormats } = useGetList(
    'transcoding',
    {
      page: 1,
      perPage: 1000,
    },
    { field: 'name', order: 'ASC' },
  )

  const formatOptions = useMemo(
    () =>
      loadingFormats
        ? []
        : Object.values(formats).map((f) => {
            return { id: f.targetFormat, name: f.name }
          }),
    [formats, loadingFormats],
  )

  const handleOriginal = useCallback(
    (original) => {
      setUseOriginalFormat(original)
      if (original) {
        setFormat(config.defaultDownsamplingFormat)
        setMaxBitRate(DEFAULT_SHARE_BITRATE)
      }
    },
    [setUseOriginalFormat, setFormat, setMaxBitRate],
  )

  const TranscodingOptionsInput = useMemo(() => {
    const Component = ({ label, basePath, ...props }) => {
      return (
        <>
          <BooleanInput
            {...props}
            source="original"
            defaultValue={originalFormat}
            label={label}
            fullWidth
            onChange={handleOriginal}
          />
          {!originalFormat && (
            <>
              <SelectInput
                {...props}
                source="format"
                defaultValue={format}
                label={translate('resources.player.fields.transcodingId')}
                choices={formatOptions}
                onChange={(event) => {
                  setFormat(event.target.value)
                }}
              />
              <SelectInput
                {...props}
                source="maxBitRate"
                label={translate('resources.player.fields.maxBitRate')}
                defaultValue={maxBitRate}
                choices={BITRATE_CHOICES}
                onChange={(event) => {
                  setMaxBitRate(event.target.value)
                }}
              />
            </>
          )}
        </>
      )
    }

    Component.displayName = 'TranscodingOptionsInput'
    return Component
  }, [
    handleOriginal,
    formatOptions,
    format,
    maxBitRate,
    originalFormat,
    translate,
  ])

  return {
    TranscodingOptionsInput,
    format,
    maxBitRate,
    originalFormat,
  }
}
