import React from 'react'
import { useTranslate } from 'react-admin'
import { ShuffleAllButton } from '../common/ShuffleAllButton'
import Notification from './Notification'
import { setSimpleMobilePref } from './simpleMobile'

// Inline styles only: MUI makeStyles/theme.palette.background.default (#303030)
// produced a blank gray screen when JSS or the ThemeProvider failed.

const shell = {
  position: 'relative',
  zIndex: 20,
  minHeight: '100vh',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 16,
  padding: 24,
  paddingBottom: 'calc(24px + env(safe-area-inset-bottom, 0px))',
  boxSizing: 'border-box',
  backgroundColor: '#121212',
  color: '#ffffff',
}

const title = {
  margin: 0,
  marginBottom: 8,
  fontSize: 28,
  fontWeight: 600,
  textAlign: 'center',
}

const actions = {
  display: 'flex',
  flexDirection: 'column',
  gap: 12,
  width: '100%',
  maxWidth: 420,
}

const fullBtn = {
  minHeight: 56,
  width: '100%',
  fontSize: 16,
  borderRadius: 8,
  border: '1px solid #666',
  background: 'transparent',
  color: '#fff',
}

const heroBtn = {
  minHeight: 72,
  width: '100%',
  fontSize: 20,
  fontWeight: 600,
  borderRadius: 8,
  border: 'none',
  background: '#90caf9',
  color: '#000',
}

class ShellErrorBoundary extends React.Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError() {
    return { hasError: true }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback
    }
    return this.props.children
  }
}

const OpenFullButton = ({ label }) => (
  <button
    type="button"
    style={fullBtn}
    onClick={() => setSimpleMobilePref(false)}
    data-testid="open-full-version"
  >
    {label}
  </button>
)

export const SimpleMobileFallback = () => (
  <div style={shell} data-testid="simple-mobile-layout">
    <h1 style={title}>Navidrome</h1>
    <div style={actions}>
      <button type="button" style={heroBtn} data-testid="shuffle-all-hero">
        Play random songs
      </button>
      <OpenFullButton label="Open full version" />
    </div>
  </div>
)

const SimpleMobileInner = () => {
  const translate = useTranslate()
  const openFull = translate('menu.openFullVersion', { _: 'Open full version' })

  return (
    <>
      <div style={shell} data-testid="simple-mobile-layout">
        <h1 style={title}>Navidrome</h1>
        <div style={actions}>
          <ShellErrorBoundary
            fallback={
              <button
                type="button"
                style={heroBtn}
                data-testid="shuffle-all-hero"
              >
                {translate('menu.playRandom', { _: 'Play random songs' })}
              </button>
            }
          >
            <ShuffleAllButton variant="hero" />
          </ShellErrorBoundary>
          <OpenFullButton label={openFull} />
        </div>
      </div>
      <Notification />
    </>
  )
}

const SimpleMobileLayout = () => (
  <ShellErrorBoundary fallback={<SimpleMobileFallback />}>
    <SimpleMobileInner />
  </ShellErrorBoundary>
)

export default SimpleMobileLayout
