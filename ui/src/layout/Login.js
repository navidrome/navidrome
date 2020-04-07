import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { Field, Form } from 'react-final-form'

import Avatar from '@material-ui/core/Avatar'
import Button from '@material-ui/core/Button'
import Card from '@material-ui/core/Card'
import CardActions from '@material-ui/core/CardActions'
import CircularProgress from '@material-ui/core/CircularProgress'
import TextField from '@material-ui/core/TextField'
import { createMuiTheme, makeStyles } from '@material-ui/core/styles'
import { ThemeProvider } from '@material-ui/styles'
import LockIcon from '@material-ui/icons/Lock'

import { Notification, useLogin, useNotify, useTranslate } from 'react-admin'

import LightTheme from '../themes/light'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  main: {
    display: 'flex',
    flexDirection: 'column',
    minHeight: '100vh',
    alignItems: 'center',
    justifyContent: 'flex-start',
    background: `url(${config.loginBackgroundURL})`,
    backgroundRepeat: 'no-repeat',
    backgroundSize: 'cover',
    backgroundPosition: 'center'
  },
  card: {
    minWidth: 300,
    marginTop: '6em'
  },
  avatar: {
    margin: '1em',
    display: 'flex',
    justifyContent: 'center'
  },
  icon: {
    backgroundColor: theme.palette.secondary.main
  },
  systemName: {
    marginTop: '1em',
    display: 'flex',
    justifyContent: 'center',
    color: 'blue' //theme.palette.grey[500]
  },
  form: {
    padding: '0 1em 1em 1em'
  },
  input: {
    marginTop: '1em'
  },
  actions: {
    padding: '0 1em 1em 1em'
  }
}))

const renderInput = ({
  meta: { touched, error } = {},
  input: { ...inputProps },
  ...props
}) => (
  <TextField
    error={!!(touched && error)}
    helperText={touched && error}
    {...inputProps}
    {...props}
    fullWidth
  />
)

const FormLogin = ({ loading, handleSubmit, validate }) => {
  const translate = useTranslate()
  const classes = useStyles()

  return (
    <Form
      onSubmit={handleSubmit}
      validate={validate}
      render={({ handleSubmit }) => (
        <form onSubmit={handleSubmit} noValidate>
          <div className={classes.main}>
            <Card className={classes.card}>
              <div className={classes.avatar}>
                <Avatar className={classes.icon}>
                  <LockIcon />
                </Avatar>
              </div>
              <div className={classes.systemName}>Navidrome</div>
              <div className={classes.form}>
                <div className={classes.input}>
                  <Field
                    autoFocus
                    name="username"
                    component={renderInput}
                    label={translate('ra.auth.username')}
                    disabled={loading}
                  />
                </div>
                <div className={classes.input}>
                  <Field
                    name="password"
                    component={renderInput}
                    label={translate('ra.auth.password')}
                    type="password"
                    disabled={loading}
                  />
                </div>
              </div>
              <CardActions className={classes.actions}>
                <Button
                  variant="contained"
                  type="submit"
                  color="primary"
                  disabled={loading}
                  className={classes.button}
                  fullWidth
                >
                  {loading && <CircularProgress size={25} thickness={2} />}
                  {translate('ra.auth.sign_in')}
                </Button>
              </CardActions>
            </Card>
            <Notification />
          </div>
        </form>
      )}
    />
  )
}

const FormSignUp = ({ loading, handleSubmit, validate }) => {
  const translate = useTranslate()
  const classes = useStyles()

  return (
    <Form
      onSubmit={handleSubmit}
      validate={validate}
      render={({ handleSubmit }) => (
        <form onSubmit={handleSubmit} noValidate>
          <div className={classes.main}>
            <Card className={classes.card}>
              <div className={classes.avatar}>
                <Avatar className={classes.icon}>
                  <LockIcon />
                </Avatar>
              </div>
              <div className={classes.systemName}>
                {translate('ra.auth.welcome1')}
              </div>
              <div className={classes.systemName}>
                {translate('ra.auth.welcome2')}
              </div>
              <div className={classes.form}>
                <div className={classes.input}>
                  <Field
                    autoFocus
                    name="username"
                    component={renderInput}
                    label={translate('ra.auth.username')}
                    disabled={loading}
                  />
                </div>
                <div className={classes.input}>
                  <Field
                    name="password"
                    component={renderInput}
                    label={translate('ra.auth.password')}
                    type="password"
                    disabled={loading}
                  />
                </div>
                <div className={classes.input}>
                  <Field
                    name="confirmPassword"
                    component={renderInput}
                    label={translate('ra.auth.confirmPassword')}
                    type="password"
                    disabled={loading}
                  />
                </div>
              </div>
              <CardActions className={classes.actions}>
                <Button
                  variant="contained"
                  type="submit"
                  color="primary"
                  disabled={loading}
                  className={classes.button}
                  fullWidth
                >
                  {loading && <CircularProgress size={25} thickness={2} />}
                  {translate('ra.auth.buttonCreateAdmin')}
                </Button>
              </CardActions>
            </Card>
            <Notification />
          </div>
        </form>
      )}
    />
  )
}
const Login = ({ location }) => {
  const [loading, setLoading] = useState(false)
  const translate = useTranslate()
  const notify = useNotify()
  const login = useLogin()

  const handleSubmit = (auth) => {
    setLoading(true)
    login(auth, location.state ? location.state.nextPathname : '/').catch(
      (error) => {
        setLoading(false)
        notify(
          typeof error === 'string'
            ? error
            : typeof error === 'undefined' || !error.message
            ? 'ra.auth.sign_in_error'
            : error.message,
          'warning'
        )
      }
    )
  }

  const validateLogin = (values) => {
    const errors = {}
    if (!values.username) {
      errors.username = translate('ra.validation.required')
    }
    if (!values.password) {
      errors.password = translate('ra.validation.required')
    }
    return errors
  }

  const validateSignup = (values) => {
    const errors = validateLogin(values)
    const regex = /^\w+$/g
    if (values.username && !values.username.match(regex)) {
      errors.username = translate('ra.validation.invalidChars')
    }
    if (!values.confirmPassword) {
      errors.confirmPassword = translate('ra.validation.required')
    }
    if (values.confirmPassword !== values.password) {
      errors.confirmPassword = translate('ra.validation.passwordDoesNotMatch')
    }
    return errors
  }

  if (config.firstTime) {
    return (
      <FormSignUp
        handleSubmit={handleSubmit}
        validate={validateSignup}
        loading={loading}
      />
    )
  }
  return (
    <FormLogin
      handleSubmit={handleSubmit}
      validate={validateLogin}
      loading={loading}
    />
  )
}

Login.propTypes = {
  authProvider: PropTypes.func,
  previousRoute: PropTypes.string
}

// We need to put the ThemeProvider decoration in another component
// Because otherwise the useStyles() hook used in Login won't get
// the right theme
const LoginWithTheme = (props) => (
  <ThemeProvider theme={createMuiTheme(LightTheme)}>
    <Login {...props} />
  </ThemeProvider>
)

export default LoginWithTheme
