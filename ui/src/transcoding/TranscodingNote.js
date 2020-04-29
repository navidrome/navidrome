import React from 'react'
import { Card, CardContent, Typography, Box } from '@material-ui/core'
import { useTranslate } from 'react-admin'

export const Interpolate = ({ message, field, children }) => {
  const split = message.split(`%{${field}}`)
  return (
    <span>
      {split[0]}
      {children}
      {split[1]}
    </span>
  )
}
export const TranscodingNote = ({ message }) => {
  const translate = useTranslate()
  return (
    <Card>
      <CardContent>
        <Typography>
          <Box fontWeight="fontWeightBold" component={'span'}>
            {translate('message.note')}:
          </Box>{' '}
          <Interpolate message={translate(message)} field={'config'}>
            <Box fontFamily="Monospace" component={'span'}>
              ND_ENABLETRANSCODINGCONFIG=true
            </Box>
          </Interpolate>
        </Typography>
      </CardContent>
    </Card>
  )
}
