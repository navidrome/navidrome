import React, { useState, useCallback, useEffect } from 'react'
import PropTypes from 'prop-types'
import { Field, Form } from 'react-final-form'
import { useDispatch } from 'react-redux'
import Button from '@material-ui/core/Button'
import Card from '@material-ui/core/Card'
import CardActions from '@material-ui/core/CardActions'
import CircularProgress from '@material-ui/core/CircularProgress'
import Link from '@material-ui/core/Link'
import TextField from '@material-ui/core/TextField'
import { ThemeProvider, makeStyles } from '@material-ui/core/styles'
import {
  createMuiTheme,
  useLogin,
  useNotify,
  useRefresh,
  useSetLocale,
  useTranslate,
  useVersion,
} from 'react-admin'
import Logo from '../icons/android-icon-192x192.png'

import Notification from './Notification'
import useCurrentTheme from '../themes/useCurrentTheme'
import config from '../config'
import { clearQueue } from '../actions'
import { retrieveTranslation } from '../i18n'
import { INSIGHTS_DOC_URL } from '../consts.js'

const useStyles = makeStyles(
  (theme) => ({
    main: {
      display: 'flex',
      flexDirection: 'column',
      minHeight: '100vh',
      alignItems: 'center',
      justifyContent: 'flex-start',
      background: `url(${config.loginBackgroundURL})`,
      backgroundRepeat: 'no-repeat',
      backgroundSize: 'cover',
      backgroundPosition: 'center',
    },
    card: {
      minWidth: 300,
      marginTop: '6em',
      overflow: 'visible',
    },
    avatar: {
      margin: '1em',
      display: 'flex',
      justifyContent: 'center',
      marginTop: '-3em',
    },
    icon: {
      backgroundColor: 'transparent',
      width: '6.3em',
      height: '6.3em',
    },
    systemName: {
      marginTop: '1em',
      display: 'flex',
      justifyContent: 'center',
      color: '#3f51b5', //theme.palette.grey[500]
    },
    welcome: {
      marginTop: '1em',
      padding: '0 1em 1em 1em',
      display: 'flex',
      justifyContent: 'center',
      flexWrap: 'wrap',
      color: '#3f51b5', //theme.palette.grey[500]
    },
    form: {
      padding: '0 1em 1em 1em',
    },
    input: {
      marginTop: '1em',
    },
    actions: {
      padding: '0 1em 1em 1em',
    },
    button: {},
    systemNameLink: {
      textDecoration: 'none',
    },
    message: {
      marginTop: '1em',
      padding: '0 1em 1em 1em',
      textAlign: 'center',
      wordBreak: 'break-word',
      fontSize: '0.875em',
    },
  }),
  { name: 'NDLogin' },
)

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

const FormLogin = ({ loading, handleSubmit, validate, onSwitchToRegister }) => {
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
                <img src={Logo} className={classes.icon} alt={'logo'} />
              </div>
              <div className={classes.systemName}>
                <a
                  href="https://www.navidrome.org"
                  target="_blank"
                  rel="noopener noreferrer"
                  className={classes.systemNameLink}
                >
                  Navidrome
                </a>
              </div>
              {config.welcomeMessage && (
                <div
                  className={classes.welcome}
                  dangerouslySetInnerHTML={{ __html: config.welcomeMessage }}
                />
              )}
              <div className={classes.form}>
                <div className={classes.input}>
                  <Field
                    autoFocus
                    name="username"
                    component={renderInput}
                    label={translate('ra.auth.username')}
                    disabled={loading}
                    spellCheck={false}
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
              {config.enableRecommendations && (
                <div className={classes.message}>
                  Don&apos;t have an account?{' '}
                  <Link
                    component="button"
                    variant="body2"
                    onClick={onSwitchToRegister}
                    style={{ cursor: 'pointer' }}
                  >
                    Sign Up
                  </Link>
                </div>
              )}
            </Card>
            <Notification />
          </div>
        </form>
      )}
    />
  )
}

const FormRegister = ({ loading, handleSubmit, onSwitchToLogin }) => {
  const classes = useStyles()

  return (
    <Form
      onSubmit={handleSubmit}
      validate={(values) => {
        const errors = {}
        if (!values.username || values.username.length < 3) {
          errors.username = 'Username must be at least 3 characters'
        }
        if (!values.email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
          errors.email = 'Valid email required'
        }
        if (!values.password || values.password.length < 6) {
          errors.password = 'Password must be at least 6 characters'
        }
        if (values.password !== values.confirmPassword) {
          errors.confirmPassword = 'Passwords do not match'
        }
        return errors
      }}
      render={({ handleSubmit }) => (
        <form onSubmit={handleSubmit} noValidate>
          <div className={classes.main}>
            <Card className={classes.card}>
              <div className={classes.avatar}>
                <img src={Logo} className={classes.icon} alt={'logo'} />
              </div>
              <div className={classes.systemName}>Create Account</div>
              <div className={classes.form}>
                <div className={classes.input}>
                  <Field
                    autoFocus
                    name="username"
                    component={renderInput}
                    label="Username"
                    disabled={loading}
                    spellCheck={false}
                  />
                </div>
                <div className={classes.input}>
                  <Field
                    name="email"
                    component={renderInput}
                    label="Email"
                    type="email"
                    disabled={loading}
                  />
                </div>
                <div className={classes.input}>
                  <Field
                    name="password"
                    component={renderInput}
                    label="Password"
                    type="password"
                    disabled={loading}
                  />
                </div>
                <div className={classes.input}>
                  <Field
                    name="confirmPassword"
                    component={renderInput}
                    label="Confirm Password"
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
                  Sign Up
                </Button>
              </CardActions>
              <div className={classes.message}>
                Already have an account?{' '}
                <Link
                  component="button"
                  variant="body2"
                  onClick={onSwitchToLogin}
                  style={{ cursor: 'pointer' }}
                >
                  Sign In
                </Link>
              </div>
            </Card>
            <Notification />
          </div>
        </form>
      )}
    />
  )
}

const InsightsNotice = ({ url }) => {
  const translate = useTranslate()
  const classes = useStyles()

  const anchorRegex = /\[(.+?)]/g
  const originalMsg = translate('ra.auth.insightsCollectionNote')

  // Split the entire message on newlines
  const lines = originalMsg.split('\n')

  const renderedLines = lines.map((line, lineIndex) => {
    const segments = []
    let lastIndex = 0
    let match

    // Find bracketed text in each line
    while ((match = anchorRegex.exec(line)) !== null) {
      // match.index is where "[something]" starts
      // match[1] is the text inside the brackets
      const bracketText = match[1]

      // Push the text before the bracket
      segments.push(line.slice(lastIndex, match.index))

      // Push the <Link> component
      segments.push(
        <Link
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          key={`${lineIndex}-${match.index}`}
          style={{ cursor: 'pointer' }}
        >
          {bracketText}
        </Link>,
      )

      // Update lastIndex to the character right after the bracketed text
      lastIndex = match.index + match[0].length
    }

    // Push the remaining text after the last bracket
    segments.push(line.slice(lastIndex))

    // Return this line’s parts, plus a <br/> if not the last line
    return (
      <React.Fragment key={lineIndex}>
        {segments}
        {lineIndex < lines.length - 1 && <br />}
      </React.Fragment>
    )
  })

  return <div className={classes.message}>{renderedLines}</div>
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
                <img src={Logo} className={classes.icon} alt={'logo'} />
              </div>
              <div className={classes.welcome}>
                {translate('ra.auth.welcome1')}
              </div>
              <div className={classes.welcome}>
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
                    spellCheck={false}
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
              <InsightsNotice url={INSIGHTS_DOC_URL} />
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
  const [showRegister, setShowRegister] = useState(false)
  const translate = useTranslate()
  const notify = useNotify()
  const login = useLogin()
  const dispatch = useDispatch()

  const handleSubmit = useCallback(
    (auth) => {
      setLoading(true)
      dispatch(clearQueue())
      login(auth, location.state ? location.state.nextPathname : '/').catch(
        (error) => {
          setLoading(false)
          notify(
            typeof error === 'string'
              ? error
              : typeof error === 'undefined' || !error.message
                ? 'ra.auth.sign_in_error'
                : error.message,
            'warning',
          )
        },
      )
    },
    [dispatch, login, notify, setLoading, location],
  )

  const handleRegister = useCallback(
    async (values) => {
      setLoading(true)
      try {
        const response = await fetch('/auth/register', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            username: values.username,
            email: values.email,
            password: values.password,
          }),
        })
        if (!response.ok) {
          const data = await response.json()
          throw new Error(data.error || 'Registration failed')
        }
        // Registration successful - auto-login
        notify('Account created! Signing in...', 'info')
        dispatch(clearQueue())
        login(
          { username: values.username, password: values.password },
          location.state ? location.state.nextPathname : '/',
        ).catch(() => {
          setLoading(false)
          setShowRegister(false)
          notify('Account created. Please sign in.', 'info')
        })
      } catch (error) {
        setLoading(false)
        notify(error.message || 'Registration failed', 'warning')
      }
    },
    [dispatch, login, notify, setLoading, location],
  )

  const validateLogin = useCallback(
    (values) => {
      const errors = {}
      if (!values.username) {
        errors.username = translate('ra.validation.required')
      }
      if (!values.password) {
        errors.password = translate('ra.validation.required')
      }
      return errors
    },
    [translate],
  )

  const validateSignup = useCallback(
    (values) => {
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
    },
    [translate, validateLogin],
  )

  if (config.firstTime) {
    return (
      <FormSignUp
        handleSubmit={handleSubmit}
        validate={validateSignup}
        loading={loading}
      />
    )
  }
  if (showRegister) {
    return (
      <FormRegister
        handleSubmit={handleRegister}
        loading={loading}
        onSwitchToLogin={() => setShowRegister(false)}
      />
    )
  }
  return (
    <FormLogin
      handleSubmit={handleSubmit}
      validate={validateLogin}
      loading={loading}
      onSwitchToRegister={() => setShowRegister(true)}
    />
  )
}

Login.propTypes = {
  authProvider: PropTypes.func,
  previousRoute: PropTypes.string,
}

// We need to put the ThemeProvider decoration in another component
// Because otherwise the useStyles() hook used in Login won't get
// the right theme
const LoginWithTheme = (props) => {
  const theme = useCurrentTheme()
  const setLocale = useSetLocale()
  const refresh = useRefresh()
  const version = useVersion()

  useEffect(() => {
    if (config.defaultLanguage !== '' && !localStorage.getItem('locale')) {
      retrieveTranslation(config.defaultLanguage)
        .then(() => {
          setLocale(config.defaultLanguage).then(() => {
            localStorage.setItem('locale', config.defaultLanguage)
          })
          refresh(true)
        })
        .catch((e) => {
          throw new Error(
            'Cannot load language "' + config.defaultLanguage + '": ' + e,
          )
        })
    }
  }, [refresh, setLocale])

  return (
    <ThemeProvider theme={createMuiTheme(theme)}>
      <Login key={version} {...props} />
    </ThemeProvider>
  )
}

export default LoginWithTheme
