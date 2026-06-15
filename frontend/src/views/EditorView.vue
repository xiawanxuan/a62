<template>
  <div class="editor-container">
    <FileTreePanel
      class="file-tree-panel"
      @select-file="handleFileSelect"
    />

    <div class="canvas-container" ref="containerRef">
      <div class="canvas-header">
        <div class="file-info">
          <span v-if="store.currentFile">
            <el-icon><Document /></el-icon>
            {{ store.currentFile.name }}
            <span class="dim">
              ({{ store.currentFile.width }} × {{ store.currentFile.height }})
            </span>
          </span>
          <span v-else class="dim">请选择声呐文件</span>
        </div>
        <div class="canvas-controls">
          <el-button-group>
            <el-button :icon="ZoomOut" @click="handleZoomOut" />
            <el-button>{{ Math.round(canvas.transform.scale * 100) }}%</el-button>
            <el-button :icon="ZoomIn" @click="handleZoomIn" />
            <el-button :icon="RefreshLeft" @click="canvas.fitToScreen">适应屏幕</el-button>
          </el-button-group>
          <el-divider direction="vertical" />
          <div class="online-users">
            <span class="dim">在线:</span>
            <span
              v-for="user in store.onlineUsers"
              :key="user.id"
              class="user-badge"
              :style="{ backgroundColor: user.color + '30', borderColor: user.color }"
              :title="user.name"
            >
              {{ user.name.charAt(0) }}
            </span>
          </div>
        </div>
      </div>

      <div class="canvas-wrapper">
        <canvas
          ref="canvasRef"
          @click="handleCanvasClick"
          @mousemove="handleCanvasMove"
          @contextmenu.prevent
        />

        <div v-if="store.isLoading" class="loading-overlay">
          <el-icon class="loading-icon" :size="48"><Loading /></el-icon>
          <p>加载中...</p>
        </div>

        <div v-if="!store.currentFile" class="empty-state">
          <el-icon :size="64" class="empty-icon"><Picture /></el-icon>
          <h3>选择左侧声呐文件开始标注</h3>
          <p>支持矩形和多边形标注，多人实时协同</p>
        </div>
      </div>
    </div>

    <AnnotationPanel
      class="annotation-panel"
      @create-annotation="handleCreateAnnotation"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick, provide } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Document, ZoomIn, ZoomOut, RefreshLeft, Loading, Picture
} from '@element-plus/icons-vue'
import { useAnnotationStore } from '@/stores/annotation'
import { useSonarCanvas } from '@/composables/useSonarCanvas'
import { useAnnotationTool } from '@/composables/useAnnotationTool'
import { useWebSocket } from '@/composables/useWebSocket'
import FileTreePanel from '@/components/FileTreePanel.vue'
import AnnotationPanel from '@/components/AnnotationPanel.vue'
import type { Annotation } from '@/types'

const store = useAnnotationStore()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const containerRef = ref<HTMLElement | null>(null)

const canvas = useSonarCanvas(canvasRef, containerRef)
const ws = useWebSocket()

const tool = useAnnotationTool(
  canvas.screenToImage,
  handleAnnotationCreated
)

provide('annotationTool', tool)

watch(() => store.currentFile, async (newFile) => {
  if (newFile) {
    try {
      await canvas.loadImage(`/api/files/${newFile.id}/image`)
      ws.connect(newFile.id, store.userId, store.userName)
    } catch (e) {
      ElMessage.error('图片加载失败')
    }
  } else {
    ws.disconnect()
  }
})

watch([() => store.annotations, tool.draft, () => store.selectedCategory], () => {
  canvas.render(
    store.annotations,
    tool.draft.value.points,
    tool.draft.value.type,
    store.selectedCategory?.color || '#ff4d4f'
  )
}, { deep: true })

ws.onAnnotationCreate.value = (ann: Annotation) => {
  store.addAnnotationFromWS(ann)
  ElMessage.info(`收到新标注: ${ann.label}`)
}

ws.onAnnotationUpdate.value = (ann: Annotation) => {
  store.updateAnnotationFromWS(ann)
}

ws.onAnnotationDelete.value = (id: string) => {
  store.deleteAnnotationFromWS(id)
}

ws.onUserJoin.value = (user) => {
  store.addOnlineUser(user)
  ElMessage.success(`${user.name} 加入协同`)
}

ws.onUserLeave.value = (userId) => {
  const user = store.onlineUsers.find(u => u.id === userId)
  store.removeOnlineUser(userId)
  if (user) {
    ElMessage.info(`${user.name} 离开`)
  }
}

function handleFileSelect(fileId: string) {
  tool.cancelDraft()
  tool.setTool(null)
  store.selectFile(fileId)
}

async function handleAnnotationCreated(data: Omit<Annotation, 'id' | 'createdAt' | 'updatedAt'>) {
  try {
    const ann = await store.createAnnotation(data)
    if (ann) {
      ws.sendAnnotationCreate(ann)
      ElMessage.success('标注创建成功')
    }
  } catch (e) {
    ElMessage.error('标注保存失败')
  }
}

function handleCanvasClick(e: MouseEvent) {
  if (!store.currentFile) return
  tool.handleCanvasClick(e)
}

function handleCanvasMove(e: MouseEvent) {
  if (!store.currentFile) return
  tool.handleCanvasMove(e)

  if (ws.isConnected.value) {
    const imgPoint = canvas.screenToImage(e.clientX, e.clientY)
    ws.sendCursorMove(imgPoint)
  }
}

function handleZoomIn() {
  if (!canvasRef.value) return
  const rect = canvasRef.value.getBoundingClientRect()
  canvas.zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, -100)
}

function handleZoomOut() {
  if (!canvasRef.value) return
  const rect = canvasRef.value.getBoundingClientRect()
  canvas.zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, 100)
}

function handleCreateAnnotation() {
}

function handleKeyDown(e: KeyboardEvent) {
  tool.handleKeyDown(e)

  if (e.key === 'Delete' && store.selectedAnnotationId) {
    ElMessageBox.confirm('确认删除该标注?', '删除确认', {
      type: 'warning'
    }).then(async () => {
      await store.deleteAnnotation(store.selectedAnnotationId!)
      ws.sendAnnotationDelete(store.selectedAnnotationId!)
      ElMessage.success('删除成功')
    }).catch(() => {})
  }
}

onMounted(async () => {
  store.initIdentity()
  await Promise.all([
    store.loadFiles(),
    store.loadCategories()
  ])

  window.addEventListener('keydown', handleKeyDown)
  nextTick(() => canvas.requestRender())
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
  ws.disconnect()
})
</script>

<style lang="scss" scoped>
.editor-container {
  display: flex;
  width: 100%;
  height: 100%;
  background: #0f0f1a;
}

.file-tree-panel {
  width: 260px;
  flex-shrink: 0;
  border-right: 1px solid #2a2a4a;
}

.canvas-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.canvas-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: #1a1a2e;
  border-bottom: 1px solid #2a2a4a;
  min-height: 56px;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;

  .dim {
    color: #6b7280;
    font-size: 12px;
  }
}

.canvas-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.online-users {
  display: flex;
  align-items: center;
  gap: 6px;

  .dim {
    color: #6b7280;
    font-size: 12px;
  }

  .user-badge {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: bold;
    border: 2px solid;
    transition: transform 0.2s;

    &:hover {
      transform: scale(1.1);
    }
  }
}

.canvas-wrapper {
  flex: 1;
  position: relative;
  overflow: hidden;
  background: #1a1a2e;
}

canvas {
  display: block;
  width: 100%;
  height: 100%;
  cursor: crosshair;
  touch-action: none;
}

.loading-overlay {
  position: absolute;
  inset: 0;
  background: rgba(15, 15, 26, 0.9);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  z-index: 10;

  .loading-icon {
    animation: spin 1s linear infinite;
    color: #409eff;
  }

  p {
    color: #9ca3af;
  }
}

.empty-state {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #6b7280;

  .empty-icon {
    color: #374151;
  }

  h3 {
    font-size: 18px;
    font-weight: 500;
    color: #9ca3af;
    margin: 0;
  }

  p {
    font-size: 14px;
    margin: 0;
  }
}

.annotation-panel {
  width: 300px;
  flex-shrink: 0;
  border-left: 1px solid #2a2a4a;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
