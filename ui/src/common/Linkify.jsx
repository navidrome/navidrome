import React, { useCallback, useMemo } from 'react'
import { Link } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import PropTypes from 'prop-types'

const useStyles = makeStyles(
  (theme) => ({
    link: {
      textDecoration: 'none',
      color: theme.palette.primary.main,
    },
  }),
  { name: 'RaLink' },
)

const Linkify = ({ text, ...rest }) => {
  const classes = useStyles()
  const linkify = useCallback((text) => {
    const urlRegex =
      /(\b(https?|ftp|file):\/\/[-A-Z0-9+&@#/%?=~_|!:,.;]*[-A-Z0-9+&@#/%=~_|])/gi
    return [...text.matchAll(urlRegex)]
  }, [])

  const parse = useCallback(() => {
    const matches = linkify(text)
    if (matches.length === 0) return text

    const elements = []
    let lastIndex = 0
    matches.forEach((match, index) => {
      // Push text located before matched string
      if (match.index > lastIndex) {
        elements.push(text.substring(lastIndex, match.index))
      }

      const href = match[0]
      // Push Link component
      elements.push(
        <Link
          {...rest}
          target="_blank"
          className={classes.link}
          rel="noopener noreferrer"
          key={index}
          href={href}
        >
          {href}
        </Link>,
      )

      lastIndex = match.index + href.length
    })

    // Push remaining text
    if (text.length > lastIndex) {
      elements.push(
        <span
          key={'last-span-key'}
          dangerouslySetInnerHTML={{ __html: text.substring(lastIndex) }}
        />,
      )
    }

    return elements.length === 1 ? elements[0] : elements
  }, [linkify, text, rest, classes.link])

  const parsedText = useMemo(() => parse(), [parse])

  return <>{parsedText}</>
}

Linkify.propTypes = {
  text: PropTypes.string,
}

export default React.memo(Linkify)
