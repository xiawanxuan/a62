import { ref, computed } from 'vue'
import type { Annotation, Point, AnnotationType, DraftAnnotation, Category } from '@/types'

export function useAnnotationTool(
  screenToImage: (x: number, y: number) => Point,
  onAnnotationCreated: (annotation: Omit<Annotation, 'id' | 'createdAt' | 'updatedAt'>) => void
) {
  const currentTool = ref<AnnotationType | null>(null)
  const selectedCategoryId = ref<string | null>(null)
  const categories = ref<Category[]>([])

  const draft = ref<DraftAnnotation>({
    type: null,
    points: [],
    categoryId: null
  })

  const isDrawing = computed(() => draft.value.type !== null)

  const selectedCategory = computed(() =>
    categories.value.find(c => c.id === selectedCategoryId.value)
  )

  const setTool = (tool: AnnotationType | null) => {
    currentTool.value = tool
    cancelDraft()
  }

  const setCategory = (categoryId: string | null) => {
    selectedCategoryId.value = categoryId
  }

  const setCategories = (cats: Category[]) => {
    categories.value = cats
    if (cats.length > 0 && !selectedCategoryId.value) {
      selectedCategoryId.value = cats[0].id
    }
  }

  const handleCanvasClick = (e: MouseEvent) => {
    if (!currentTool.value || !selectedCategoryId.value) return

    const point = screenToImage(e.clientX, e.clientY)

    if (currentTool.value === 'rectangle') {
      if (draft.value.points.length === 0) {
        draft.value = {
          type: 'rectangle',
          points: [point, point],
          categoryId: selectedCategoryId.value
        }
      } else {
        draft.value.points[1] = point
        finishRectangle()
      }
    } else if (currentTool.value === 'polygon') {
      if (draft.value.type !== 'polygon') {
        draft.value = {
          type: 'polygon',
          points: [point],
          categoryId: selectedCategoryId.value
        }
      } else {
        const firstPoint = draft.value.points[0]
        const dist = Math.hypot(point.x - firstPoint.x, point.y - firstPoint.y)
        if (dist < 20 && draft.value.points.length >= 3) {
          finishPolygon()
        } else {
          draft.value.points.push(point)
        }
      }
    }
  }

  const handleCanvasMove = (e: MouseEvent) => {
    if (!currentTool.value || draft.value.points.length === 0) return

    const point = screenToImage(e.clientX, e.clientY)

    if (currentTool.value === 'rectangle' && draft.value.points.length >= 2) {
      draft.value.points[1] = point
    }
  }

  const finishRectangle = () => {
    if (draft.value.points.length < 2 || !selectedCategory.value) return

    const [p1, p2] = draft.value.points
    const width = Math.abs(p2.x - p1.x)
    const height = Math.abs(p2.y - p1.y)

    if (width < 5 || height < 5) {
      cancelDraft()
      return
    }

    const category = selectedCategory.value
    onAnnotationCreated({
      fileId: '',
      type: 'rectangle',
      points: [p1, p2],
      categoryId: category.id,
      label: category.name,
      color: category.color,
      createdBy: 'user'
    })

    cancelDraft()
  }

  const finishPolygon = () => {
    if (draft.value.points.length < 3 || !selectedCategory.value) return

    const category = selectedCategory.value
    onAnnotationCreated({
      fileId: '',
      type: 'polygon',
      points: [...draft.value.points],
      categoryId: category.id,
      label: category.name,
      color: category.color,
      createdBy: 'user'
    })

    cancelDraft()
  }

  const cancelDraft = () => {
    draft.value = {
      type: null,
      points: [],
      categoryId: null
    }
  }

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      cancelDraft()
      setTool(null)
    } else if (e.key === 'r' || e.key === 'R') {
      setTool('rectangle')
    } else if (e.key === 'p' || e.key === 'P') {
      setTool('polygon')
    } else if (e.key === 'Enter' && draft.value.type === 'polygon' && draft.value.points.length >= 3) {
      finishPolygon()
    }
  }

  return {
    currentTool,
    selectedCategoryId,
    selectedCategory,
    categories,
    draft,
    isDrawing,
    setTool,
    setCategory,
    setCategories,
    handleCanvasClick,
    handleCanvasMove,
    cancelDraft,
    handleKeyDown
  }
}
