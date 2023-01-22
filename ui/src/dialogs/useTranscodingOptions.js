import React, { useCallback, useMemo, useState } from 'react'
import config from '../config'
import { DEFAULT_SHARE_BITRATE } from '../consts'
import { BooleanInput, SelectInput, useGetList } from 'react-admin'

export const useTranscodingOptions = () => {
  const [format, setFormat] = useState(config.defaultDownsamplingFormat)
  const [maxBitRate, setMaxBitRate] = useState(DEFAULT_SHARE_BITRATE)
  const [originalFormat, setUseOriginalFormat] = useState(true)

  const { data: formats, loading: loadingFormats } = useGetList(
    'transcoding',
    {
      page: 1,
      perPage: 1000,
    },
    { field: 'name', order: 'ASC' }
  )

  const formatOptions = useMemo(
    () =>
      loadingFormats
        ? []
        : Object.values(formats).map((f) => {
            return { id: f.targetFormat, name: f.targetFormat }
          }),
    [formats, loadingFormats]
  )

  const handleOriginal = useCallback(
    (original) => {
      setUseOriginalFormat(original)
      if (original) {
        setFormat(config.defaultDownsamplingFormat)
        setMaxBitRate(DEFAULT_SHARE_BITRATE)
      }
    },
    [setUseOriginalFormat, setFormat, setMaxBitRate]
  )

  const TranscodingOptionsInput = useMemo(() => {
    return ({ basePath, ...props }) => {
      return (
        <>
          <BooleanInput
            {...props}
            source="original"
            defaultValue={originalFormat}
            label={'Share in original format'}
            onChange={handleOriginal}
          />
          {!originalFormat && (
            <>
              <SelectInput
                {...props}
                source="format"
                defaultValue={format}
                choices={formatOptions}
                onChange={(event) => {
                  setFormat(event.target.value)
                }}
              />
              <SelectInput
                {...props}
                source="maxBitRate"
                defaultValue={maxBitRate}
                choices={[
                  { id: 32, name: '32' },
                  { id: 48, name: '48' },
                  { id: 64, name: '64' },
                  { id: 80, name: '80' },
                  { id: 96, name: '96' },
                  { id: 112, name: '112' },
                  { id: 128, name: '128' },
                  { id: 160, name: '160' },
                  { id: 192, name: '192' },
                  { id: 256, name: '256' },
                  { id: 320, name: '320' },
                ]}
                onChange={(event) => {
                  setMaxBitRate(event.target.value)
                }}
              />
            </>
          )}
        </>
      )
    }
  }, [handleOriginal, formatOptions, format, maxBitRate, originalFormat])

  return {
    TranscodingOptionsInput,
    format,
    maxBitRate,
    originalFormat,
  }
}
