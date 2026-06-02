import React from 'react'
import { Link } from 'react-router-dom'
import { Breadcrumbs as MuiBreadcrumbs, Typography } from '@material-ui/core'
import NavigateNextIcon from '@material-ui/icons/NavigateNext'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'

const useStyles = makeStyles((theme) => ({
  root: {
    marginBottom: theme.spacing(2),
  },
  link: {
    cursor: 'pointer',
    color: theme.palette.text.secondary,
    textDecoration: 'none',
    '&:hover': {
      textDecoration: 'underline',
    },
  },
  current: {
    color: theme.palette.text.primary,
  },
}))

const Breadcrumbs = ({ breadcrumbs }) => {
  const classes = useStyles()
  const translate = useTranslate()

  if (!breadcrumbs || !Array.isArray(breadcrumbs)) return null

  return (
    <MuiBreadcrumbs
      separator={<NavigateNextIcon fontSize="small" />}
      aria-label="breadcrumb"
      className={classes.root}
    >
      <Link to="/folder" className={classes.link}>
        {translate('menu.folders')}
      </Link>
      {breadcrumbs.map((b, index) => {
        if (!b || !b.id || !b.name) return null
        const isLast = index === breadcrumbs.length - 1
        // For libraries (first breadcrumb), we might want to navigate back to the root list if it's the only one,
        // but let's just keep it simple.
        return isLast ? (
          <Typography key={b.id} className={classes.current}>
            {b.name}
          </Typography>
        ) : (
          <Link key={b.id} to={`/folder/${b.id}`} className={classes.link}>
            {b.name}
          </Link>
        )
      })}
    </MuiBreadcrumbs>
  )
}

export default Breadcrumbs
