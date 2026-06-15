import { ref, shallowRef, onMounted, onUnmounted, watch } from 'vue'
import type { Annotation, Point, ViewTransform } from '@/types'

export function useSonarCanvas(
  canvasRef: ReturnType<typeof ref<HTMLCanvasElement | null>>,
  containerRef: ReturnType<typeof ref<HTMLElement | null>>
) {
  const ctx = shallowRef<CanvasRenderingContext2D | null>(null)
  const image = shallowRef<HTMLImageElement | null>(null)
  const imageLoaded = ref(false)
  const imageSize = ref({ width: 0, height: 0 })

  const transform = ref<ViewTransform>({
    scale: 1,
    offsetX: 0,
    offsetY: 0
  })

  const minScale = 0.1
  const maxScale = 5
  const wheelSensitivity = 0.001

  let isDragging = false
  let lastMousePos: Point = { x: 0, y: 0 }
  let animationFrameId: number | null = null
  let needsRender = false

  let canvasRectCache: { left: number; top: number; width: number; height: number } | null = null
  let canvasRectCacheTime = 0
  const RECT_CACHE_TTL = 16

  const getCanvasRect = () => {
    const now = Date.now()
    if (!canvasRectCache || now - canvasRectCacheTime > RECT_CACHE_TTL) {
      const el = canvasRef.value
      if (el) {
        const rect = el.getBoundingClientRect()
        canvasRectCache = { left: rect.left, top: rect.top, width: rect.width, height: rect.height }
        canvasRectCacheTime = now
      }
    }
    return canvasRectCache || { left: 0, top: 0, width: 0, height: 0 }
  }

  const invalidateCanvasRect = () => {
    canvasRectCache = null
  }

  const loadImage = async (url: string) => {
    imageLoaded.value = false
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.decoding = 'async'

    return new Promise<void>((resolve, reject) => {
      img.onload = () => {
        image.value = img
        imageSize.value = { width: img.width, height: img.height }
        fitToScreen()
        imageLoaded.value = true
        requestRender()
        resolve()
      }
      img.onerror = reject
      img.src = url
    })
  }

  const fitToScreen = () => {
    if (!containerRef.value || !image.value) return

    const container = containerRef.value
    const scaleX = container.clientWidth / image.value.width
    const scaleY = container.clientHeight / image.value.height
    const scale = Math.min(scaleX, scaleY, 1)

    transform.value = {
      scale,
      offsetX: (container.clientWidth - image.value.width * scale) / 2,
      offsetY: (container.clientHeight - image.value.height * scale) / 2
    }
    requestRender()
  }

  const viewportToCanvas = (viewportX: number, viewportY: number): Point => {
    const rect = getCanvasRect()
    return {
      x: viewportX - rect.left,
      y: viewportY - rect.top
    }
  }

  const screenToImage = (screenX: number, screenY: number): Point => {
    const canvasPoint = viewportToCanvas(screenX, screenY)
    return canvasToImage(canvasPoint.x, canvasPoint.y)
  }

  const canvasToImage = (canvasX: number, canvasY: number): Point => {
    return {
      x: (canvasX - transform.value.offsetX) / transform.value.scale,
      y: (canvasY - transform.value.offsetY) / transform.value.scale
    }
  }

  const imageToScreen = (imageX: number, imageY: number): Point => {
    return imageToCanvas(imageX, imageY)
  }

  const imageToCanvas = (imageX: number, imageY: number): Point => {
    return {
      x: imageX * transform.value.scale + transform.value.offsetX,
      y: imageY * transform.value.scale + transform.value.offsetY
    }
  }

  const zoomAt = (screenX: number, screenY: number, delta: number) => {
    const canvasPoint = viewportToCanvas(screenX, screenY)
    if (canvasPoint.x < 0 || canvasPoint.y < 0) return

    const imagePoint = canvasToImage(canvasPoint.x, canvasPoint.y)

    const scaleFactor = Math.exp(-delta * wheelSensitivity)
    let newScale = transform.value.scale * scaleFactor
    newScale = Math.max(minScale, Math.min(maxScale, newScale))

    transform.value.offsetX = canvasPoint.x - imagePoint.x * newScale
    transform.value.offsetY = canvasPoint.y - imagePoint.y * newScale
    transform.value.scale = newScale

    requestRender()
  }

  const pan = (dx: number, dy: number) => {
    transform.value.offsetX += dx
    transform.value.offsetY += dy
    requestRender()
  }

  const getCanvasCssSize = () => {
    const container = containerRef.value
    if (!container) return { width: 0, height: 0 }
    return { width: container.clientWidth, height: container.clientHeight }
  }

  const render = (annotations: Annotation[], draftPoints: Point[] = [], draftType: string | null = null, selectedCategoryColor: string = '#ff4d4f') => {
    if (!ctx.value || !canvasRef.value) return

    const c = ctx.value
    const { width: cssWidth, height: cssHeight } = getCanvasCssSize()

    c.save()
    c.setTransform(1, 0, 0, 1, 0, 0)
    c.clearRect(0, 0, canvasRef.value.width, canvasRef.value.height)
    c.fillStyle = '#1a1a2e'
    c.fillRect(0, 0, canvasRef.value.width, canvasRef.value.height)
    c.restore()

    if (image.value && imageLoaded.value) {
      c.save()
      c.imageSmoothingEnabled = transform.value.scale < 1
      c.imageSmoothingQuality = transform.value.scale < 1 ? 'high' : 'medium'
      c.drawImage(
        image.value,
        transform.value.offsetX,
        transform.value.offsetY,
        image.value.width * transform.value.scale,
        image.value.height * transform.value.scale
      )
      c.restore()
    }

    annotations.forEach(ann => renderAnnotation(c, ann))

    if (draftPoints.length > 0 && draftType) {
      renderDraftAnnotation(c, draftPoints, draftType, selectedCategoryColor)
    }
  }

  const renderAnnotation = (c: CanvasRenderingContext2D, annotation: Annotation) => {
    const points = annotation.points.map(p => imageToCanvas(p.x, p.y))

    c.save()
    c.strokeStyle = annotation.color
    c.fillStyle = annotation.color + '30'
    c.lineWidth = 2
    c.lineJoin = 'round'

    if (annotation.type === 'rectangle' && points.length >= 2) {
      const [p1, p2] = points
      const x = Math.min(p1.x, p2.x)
      const y = Math.min(p1.y, p2.y)
      const w = Math.abs(p2.x - p1.x)
      const h = Math.abs(p2.y - p1.y)

      c.strokeRect(x, y, w, h)
      c.fillRect(x, y, w, h)
    } else if (annotation.type === 'polygon' && points.length >= 3) {
      c.beginPath()
      c.moveTo(points[0].x, points[0].y)
      for (let i = 1; i < points.length; i++) {
        c.lineTo(points[i].x, points[i].y)
      }
      c.closePath()
      c.stroke()
      c.fill()

      points.forEach(p => {
        c.beginPath()
        c.arc(p.x, p.y, 4, 0, Math.PI * 2)
        c.fillStyle = annotation.color
        c.fill()
      })
    }

    if (points.length > 0) {
      c.fillStyle = annotation.color
      c.font = 'bold 12px sans-serif'
      c.fillText(annotation.label, points[0].x + 5, points[0].y - 5)
    }

    c.restore()
  }

  const renderDraftAnnotation = (c: CanvasRenderingContext2D, points: Point[], type: string, color: string) => {
    const screenPoints = points.map(p => imageToCanvas(p.x, p.y))

    c.save()
    c.strokeStyle = color
    c.fillStyle = color + '20'
    c.lineWidth = 2
    c.setLineDash([5, 5])

    if (type === 'rectangle' && screenPoints.length >= 2) {
      const [p1, p2] = screenPoints
      const x = Math.min(p1.x, p2.x)
      const y = Math.min(p1.y, p2.y)
      const w = Math.abs(p2.x - p1.x)
      const h = Math.abs(p2.y - p1.y)
      c.strokeRect(x, y, w, h)
      c.fillRect(x, y, w, h)
    } else if (type === 'polygon' && screenPoints.length >= 1) {
      c.beginPath()
      c.moveTo(screenPoints[0].x, screenPoints[0].y)
      for (let i = 1; i < screenPoints.length; i++) {
        c.lineTo(screenPoints[i].x, screenPoints[i].y)
      }
      if (screenPoints.length >= 3) {
        c.closePath()
        c.stroke()
        c.fill()
      } else {
        c.stroke()
      }

      screenPoints.forEach(p => {
        c.beginPath()
        c.arc(p.x, p.y, 5, 0, Math.PI * 2)
        c.fillStyle = color
        c.fill()
        c.strokeStyle = '#fff'
        c.lineWidth = 1
        c.setLineDash([])
        c.stroke()
        c.setLineDash([5, 5])
      })
    }

    c.restore()
  }

  const requestRender = () => {
    needsRender = true
    if (animationFrameId === null) {
      animationFrameId = requestAnimationFrame(renderLoop)
    }
  }

  const renderLoop = () => {
    if (needsRender) {
      needsRender = false
    }
    animationFrameId = requestAnimationFrame(renderLoop)
  }

  const handleMouseDown = (e: MouseEvent) => {
    if (e.button === 1 || (e.button === 0 && e.altKey)) {
      isDragging = true
      lastMousePos = { x: e.clientX, y: e.clientY }
      if (canvasRef.value) {
        canvasRef.value.style.cursor = 'grabbing'
      }
    }
  }

  const handleMouseMove = (e: MouseEvent) => {
    if (isDragging) {
      const dx = e.clientX - lastMousePos.x
      const dy = e.clientY - lastMousePos.y
      pan(dx, dy)
      lastMousePos = { x: e.clientX, y: e.clientY }
    }
  }

  const handleMouseUp = () => {
    isDragging = false
    if (canvasRef.value) {
      canvasRef.value.style.cursor = 'crosshair'
    }
  }

  const handleWheel = (e: WheelEvent) => {
    e.preventDefault()
    zoomAt(e.clientX, e.clientY, e.deltaY)
  }

  const handleTouchStart = (e: TouchEvent) => {
    if (e.touches.length === 2) {
      e.preventDefault()
    }
  }

  const handlePinch = (e: TouchEvent) => {
    if (e.touches.length !== 2) return
    e.preventDefault()

    const t1 = e.touches[0]
    const t2 = e.touches[1]
    const centerX = (t1.clientX + t2.clientX) / 2
    const centerY = (t1.clientY + t2.clientY) / 2
    const distance = Math.hypot(t2.clientX - t1.clientX, t2.clientY - t1.clientY)

    if (!(window as any)._lastPinchDistance) {
      (window as any)._lastPinchDistance = distance
      return
    }

    const delta = (window as any)._lastPinchDistance - distance
    ;(window as any)._lastPinchDistance = distance
    zoomAt(centerX, centerY, delta * 2)
  }

  const resize = () => {
    if (!canvasRef.value || !containerRef.value || !ctx.value) return

    const container = containerRef.value
    const dpr = window.devicePixelRatio || 1

    canvasRef.value.width = container.clientWidth * dpr
    canvasRef.value.height = container.clientHeight * dpr
    canvasRef.value.style.width = container.clientWidth + 'px'
    canvasRef.value.style.height = container.clientHeight + 'px'

    ctx.value.setTransform(dpr, 0, 0, dpr, 0, 0)
    invalidateCanvasRect()
    requestRender()
  }

  const init = () => {
    if (!canvasRef.value) return
    ctx.value = canvasRef.value.getContext('2d')
    resize()

    window.addEventListener('resize', resize)
    canvasRef.value.addEventListener('wheel', handleWheel, { passive: false })
    canvasRef.value.addEventListener('mousedown', handleMouseDown)
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    canvasRef.value.addEventListener('touchstart', handleTouchStart, { passive: false })
    canvasRef.value.addEventListener('touchmove', handlePinch, { passive: false })
  }

  const destroy = () => {
    if (animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId)
    }
    window.removeEventListener('resize', resize)
    if (canvasRef.value) {
      canvasRef.value.removeEventListener('wheel', handleWheel)
      canvasRef.value.removeEventListener('mousedown', handleMouseDown)
      canvasRef.value.removeEventListener('touchstart', handleTouchStart)
      canvasRef.value.removeEventListener('touchmove', handlePinch)
    }
    window.removeEventListener('mousemove', handleMouseMove)
    window.removeEventListener('mouseup', handleMouseUp)
  }

  watch(transform, () => requestRender(), { deep: true })

  onMounted(init)
  onUnmounted(destroy)

  return {
    loadImage,
    fitToScreen,
    screenToImage,
    imageToScreen,
    render,
    requestRender,
    transform,
    imageLoaded,
    imageSize,
    handleMouseDown,
    handleMouseMove,
    handleMouseUp
  }
}
