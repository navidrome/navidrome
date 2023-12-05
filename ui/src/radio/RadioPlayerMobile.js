import clsx from 'clsx'
import React, { memo } from 'react'
import RadioTitle from './RadioTitle'

const prefix = 'react-jinke-music-player-mobile'

const RadioPlayerMobile = ({
  cover,
  icon,
  id,
  loading,
  locale,
  metadata,
  name,
  onClose,
  onCoverClick,
  onFix,
  onPlay,
  playing,
}) => (
  <div className={clsx(prefix, 'default-bg')}>
    <div className={`${prefix}-header group`}>
      <div className={`${prefix}-header-title`} title={name}>
        <RadioTitle
          id={id}
          isMobile={true}
          metadata={metadata}
          name={name}
          onFix={onFix}
        />
      </div>

      <div className={`${prefix}-header-right`} onClick={onClose}>
        {icon.close}
      </div>
    </div>

    {cover && (
      <div
        className={`${prefix}-cover text-center`}
        onClick={() => onCoverClick()}
      >
        <img
          src={cover}
          alt="cover"
          className={clsx('cover', {
            'img-rotate-pause': !playing || !cover,
          })}
        />
      </div>
    )}

    <div className={`${prefix}-toggle text-center group`}>
      {loading ? (
        <span className="group loading-icon">{icon.loading}</span>
      ) : (
        <span
          className="group play-btn"
          title={playing ? locale.clickToPauseText : locale.clickToPlayText}
          onClick={onPlay}
        >
          {playing ? icon.pause : icon.play}
        </span>
      )}
    </div>
  </div>
)

RadioPlayerMobile.defaultProps = {
  icon: {},
  renderAudioTitle: () => {},
}

export default memo(RadioPlayerMobile)
