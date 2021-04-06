// import butterchurn from 'butterchurn'
// import butterchurnPresets from 'butterchurn-presets'

// import isButterchurnSupported from 'butterchurn/lib/isSupported.min'

// if (isButterchurnSupported()) {
//   // Load and use butterchurn
// }

// // initialize audioContext and get canvas

// const visualizer = butterchurn.createVisualizer(audioContext, canvas, {
//   width: 800,
//   height: 600,
// })

// // get audioNode from audio source or microphone

// visualizer.connectAudio(audioNode)

// // load a preset

// const presets = butterchurnPresets.getPresets()
// const preset =
//   presets['Flexi, martin + geiss - dedicated to the sherwin maxawow']

// visualizer.loadPreset(preset, 0.0) // 2nd argument is the number of seconds to blend presets

// // resize visualizer

// visualizer.setRendererSize(1600, 1200)

// // render a frame

// visualizer.render()

import React, { useEffect, useLayoutEffect, useRef, useState } from 'react'
import ReactDOM from 'react-dom'
// import { Location } from 'history';
import butterchurn from 'butterchurn'
import butterchurnPresets from 'butterchurn-presets'
import _ from 'lodash'

export type VisualizerProps = {
  audioContext?: AudioContext
  previousNode?: AudioNode
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location?: Location<any>
  trackName?: string
  presetName: string
}

type Size = {
  x: number
  y: number
}

type ButterchurnVisualizer = {
  render: () => void
  setRendererSize: (x: number, y: number) => void
  connectAudio: (audioNode: AudioNode) => void
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  loadPreset: (preset: Record<string, any>, blendTime: number) => void
  launchSongTitleAnim: (title: string) => void
}

const MilkDropVisualizer: React.FC<VisualizerProps> = ({
  audioContext,
  previousNode,
  location,
  trackName,
  presetName,
}) => {
  const canvasRef = useRef()
  const [canvasSize, setCanvasSize] = useState<Size>({ x: 1, y: 1 })
  const [visualizerNode, setVisualizerNode] = useState<HTMLElement>()
  const [visualizer, setVisualizer] = useState<ButterchurnVisualizer>()

  useEffect(() => {
    if (canvasRef.current) {
      const _visualizer = butterchurn.createVisualizer(
        audioContext,
        canvasRef.current,
        {
          width: canvasSize.x,
          height: canvasSize.y,
          textureRatio: 1,
        }
      )
      _visualizer.connectAudio(previousNode)
      setVisualizer(_visualizer)
    }
  }, [canvasRef, canvasSize, audioContext, previousNode])

  useEffect(() => {
    if (visualizer) {
      visualizer.setRendererSize(canvasSize.x, canvasSize.y)
    }
  }, [visualizer, canvasSize])

  useLayoutEffect(() => {
    const visualizerNodeElement = document.getElementById('visualizer_node')
    setVisualizerNode(visualizerNodeElement)
  }, [location])

  useEffect(() => {
    if (!previousNode || !visualizer) {
      return
    }

    const presets = butterchurnPresets.getPresets()
    const preset = presets[presetName]
    visualizer?.loadPreset(preset, 0.0)

    let animationFrameRequest: number | null = null
    const renderingLoop = () => {
      visualizer?.render()
      animationFrameRequest = requestAnimationFrame(renderingLoop)
    }
    renderingLoop()

    return () => {
      if (animationFrameRequest !== null) {
        cancelAnimationFrame(animationFrameRequest)
      }
    }
  }, [previousNode, visualizerNode, visualizer, presetName])

  useEffect(() => {
    if (!visualizer || !trackName) {
      return
    }
    visualizer.launchSongTitleAnim(trackName)
  }, [visualizer, trackName])

  const onResize = _.debounce(
    (contentRect) =>
      setCanvasSize({
        x: contentRect.bounds.width,
        y: contentRect.bounds.height,
      }),
    2000
  )

  return visualizerNode
    ? ReactDOM.createPortal(
        previousNode && visualizerNode && (
          <Measure bounds onResize={onResize}>
            {({ measureRef }) => (
              <div className={styles.visualizer} ref={measureRef}>
                <canvas
                  ref={canvasRef}
                  width={canvasSize.x}
                  height={canvasSize.y}
                />
              </div>
            )}
          </Measure>
        ),
        visualizerNode
      )
    : null
}

export default MilkDropVisualizer
