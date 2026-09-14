import React from 'react'
import { useTranslate } from 'react-admin'
import { ShuffleAllButton } from '../common/ShuffleAllButton'
import Notification from './Notification'
import { setSimpleMobilePref } from './simpleMobile'

const shell = {
  position: 'relative',
  zIndex: 10,
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

const SimpleMobileLayout = () => {
  const translate = useTranslate()

  return (
    <>
      <div style={shell} data-testid="simple-mobile-layout">
        <h1 style={title}>Navidrome</h1>
        <div style={actions}>
          <ShuffleAllButton variant="hero" />
          <button
            type="button"
            style={fullBtn}
            onClick={() => setSimpleMobilePref(false)}
            data-testid="open-full-version"
          >
            {translate('menu.openFullVersion', { _: 'Open full version' })}
          </button>
        </div>
      </div>
      <Notification />
    </>
  )
}

export default SimpleMobileLayout
