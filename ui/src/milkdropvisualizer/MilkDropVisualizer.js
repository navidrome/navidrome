import React, { useRef, useEffect, useState } from 'react'
import { useSelector } from 'react-redux'
import butterchurn from 'butterchurn'
import butterchurnPresets from 'butterchurn-presets'
import isButterchurnSupported from 'butterchurn/lib/isSupported.min'

const MilkDropVisualizer = () => {
  const songInfo = useSelector((state) => state.queue.queue[0])
  const [audioContext] = useState(new AudioContext())
  const [visualizer, setVisualizer] = useState()

  const canvasRef = useRef(null)
  const audioRef = useRef(document.createElement('AUDIO'))
  audioRef.current.src = songInfo.musicSrc

  useEffect(() => {
    if (canvasRef.current) {
      const _visualizer = butterchurn.createVisualizer(
        audioContext,
        canvasRef.current,
        {
          width: canvasRef.current.clientWidth,
          height: canvasRef.current.clientHeight,
          textureRatio: 1,
        }
      )
      _visualizer.connectAudio(
        audioContext.createMediaElementSource(audioRef.current)
      )
      setVisualizer(_visualizer)
    }
  }, [canvasRef, audioContext])

  useEffect(() => {
    if (!audioRef.current || !visualizer) {
      return
    }

    const presets = butterchurnPresets.getPresets()
    const preset =
      presets['Flexi, martin + geiss - dedicated to the sherwin maxawow']

    visualizer.loadPreset(preset, 0.0)

    let animationFrameRequest = null
    const renderingLoop = () => {
      visualizer.render()
      animationFrameRequest = requestAnimationFrame(renderingLoop)
    }
    renderingLoop()

    return () => {
      if (animationFrameRequest !== null) {
        cancelAnimationFrame(animationFrameRequest)
      }
    }
  }, [audioRef, visualizer])

  useEffect(() => {
    if (!visualizer || !songInfo.name) {
      return
    }
    visualizer.launchSongTitleAnim(songInfo.name)
  }, [visualizer, songInfo.name])

  return (
    <>
      <canvas
        ref={canvasRef}
        style={{ height: '100%', width: '100%', backgroundColor: 'inherit' }}
      ></canvas>
    </>
  )
}

const MilkDropVisualizerFallback = () => {
  return isButterchurnSupported() ? (
    <MilkDropVisualizer />
  ) : (
    <div>Broswer does not support WebGL </div>
  )
}

export default MilkDropVisualizerFallback
