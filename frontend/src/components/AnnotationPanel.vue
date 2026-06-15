<template>
  <div class="annotation-panel">
    <div class="panel-header">
      <h3><el-icon><Edit /></el-icon> 标注工具</h3>
      <div class="header-user">
        <el-tooltip content="修改昵称" placement="left">
          <el-input
            v-model="userNameInput"
            class="name-input"
            size="small"
            @blur="commitUserName"
            @keyup.enter="commitUserName"
          />
        </el-tooltip>
      </div>
    </div>

    <div class="tools-section">
      <div class="section-title">绘图工具</div>
      <div class="tool-buttons">
        <el-button
          :type="tool.currentTool.value === 'rectangle' ? 'primary' : 'default'"
          :icon="Grid"
          @click="tool.setTool('rectangle')"
        >
          矩形 (R)
        </el-button>
        <el-button
          :type="tool.currentTool.value === 'polygon' ? 'primary' : 'default'"
          :icon="Operation"
          @click="tool.setTool('polygon')"
        >
          多边形 (P)
        </el-button>
        <el-button
          v-if="tool.isDrawing.value"
          :icon="Close"
          type="danger"
          @click="handleCancel"
        >
          取消 (Esc)
        </el-button>
      </div>

      <div v-if="tool.currentTool.value === 'polygon'" class="tip">
        <el-icon><InfoFilled /></el-icon>
        点击添加顶点，点击起点或按 Enter 闭合
      </div>
    </div>

    <div class="section">
      <div class="section-title">
        <span>目标分类</span>
        <el-button
          :icon="Setting"
          size="small"
          text
          type="primary"
          @click="categoryDialogVisible = true"
        >
          管理
        </el-button>
      </div>

      <div class="category-group" v-if="store.globalCategories.length > 0">
        <div class="group-label">内置</div>
        <div class="category-list">
          <div
            v-for="cat in store.globalCategories"
            :key="cat.id"
            class="category-item"
            :class="{ active: store.selectedCategoryId === cat.id }"
            @click="store.setSelectedCategory(cat.id)"
          >
            <span class="color-dot" :style="{ backgroundColor: cat.color }" />
            <span class="cat-name">{{ cat.name }}</span>
            <span class="cat-count">
              {{ (store.annotationsByCategory[cat.id] || []).length }}
            </span>
          </div>
        </div>
      </div>

      <div class="category-group" v-if="store.userCategories.length > 0">
        <div class="group-label">
          我的
          <el-tag size="small" type="success" effect="plain">{{ store.userCategories.length }}</el-tag>
        </div>
        <div class="category-list">
          <div
            v-for="cat in store.userCategories"
            :key="cat.id"
            class="category-item"
            :class="{ active: store.selectedCategoryId === cat.id }"
            @click="store.setSelectedCategory(cat.id)"
          >
            <span class="color-dot" :style="{ backgroundColor: cat.color }" />
            <span class="cat-name">{{ cat.name }}</span>
            <span class="cat-count">
              {{ (store.annotationsByCategory[cat.id] || []).length }}
            </span>
          </div>
        </div>
      </div>

      <div v-if="store.categories.length === 0 && !store.isCategoriesLoading" class="empty-annotations">
        暂无分类，点击右上角"管理"添加
      </div>
      <div v-if="store.isCategoriesLoading" class="empty-annotations">
        加载中...
      </div>
    </div>

    <div class="section">
      <div class="section-title">
        <span>标注列表</span>
        <el-tag size="small" type="info">{{ store.annotations.length }}</el-tag>
      </div>

      <div class="annotation-list">
        <div
          v-for="ann in store.annotations"
          :key="ann.id"
          class="annotation-item"
          :class="{ selected: store.selectedAnnotationId === ann.id }"
          @click="store.setSelectedAnnotation(ann.id)"
        >
          <span class="ann-color" :style="{ backgroundColor: ann.color }" />
          <div class="ann-info">
            <div class="ann-label">{{ ann.label }}</div>
            <div class="ann-meta">{{ ann.type === 'rectangle' ? '矩形' : '多边形' }}</div>
          </div>
          <el-button
            :icon="Delete"
            size="small"
            text
            type="danger"
            @click.stop="handleDelete(ann.id)"
          />
        </div>

        <div v-if="store.annotations.length === 0" class="empty-annotations">
          暂无标注
        </div>
      </div>
    </div>

    <div class="section">
      <div class="section-title">
        <span>版本历史</span>
        <el-tag size="small" type="info">{{ store.snapshots.length }}/30</el-tag>
      </div>

      <div class="snapshot-list">
        <div
          v-for="snap in store.snapshots.slice(0, 10)"
          :key="snap.id"
          class="snapshot-item"
        >
          <div class="snap-info">
            <div class="snap-message">{{ snap.message }}</div>
            <div class="snap-time">{{ formatTime(snap.createdAt) }}</div>
          </div>
          <el-button
            :icon="Refresh"
            size="small"
            @click="handleRestore(snap.id)"
          >
            回滚
          </el-button>
        </div>

        <div v-if="store.snapshots.length === 0" class="empty-annotations">
          暂无快照
        </div>
      </div>
    </div>

    <div class="section">
      <div class="section-title">键盘快捷键</div>
      <div class="shortcuts">
        <div class="shortcut-item"><kbd>R</kbd> 矩形工具</div>
        <div class="shortcut-item"><kbd>P</kbd> 多边形工具</div>
        <div class="shortcut-item"><kbd>Esc</kbd> 取消绘制</div>
        <div class="shortcut-item"><kbd>Delete</kbd> 删除选中</div>
        <div class="shortcut-item"><kbd>滚轮</kbd> 缩放</div>
        <div class="shortcut-item"><kbd>Alt+拖动</kbd> 平移</div>
      </div>
    </div>

    <CategoryManagerDialog v-model="categoryDialogVisible" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, inject } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Edit, Grid, Operation, Close, InfoFilled, Delete, Refresh, Setting
} from '@element-plus/icons-vue'
import { useAnnotationStore } from '@/stores/annotation'
import type { Annotation } from '@/types'
import CategoryManagerDialog from '@/components/CategoryManagerDialog.vue'
import type { useAnnotationTool } from '@/composables/useAnnotationTool'

type AnnotationToolType = ReturnType<typeof useAnnotationTool>

const emit = defineEmits<{
  createAnnotation: []
}>()

const store = useAnnotationStore()

const tool = inject('annotationTool') as AnnotationToolType

watch(() => store.categories, (cats) => {
  tool.setCategories(cats)
}, { immediate: true, deep: true })

watch(() => store.selectedCategoryId, (id) => {
  tool.setCategory(id)
}, { immediate: true })

const categoryDialogVisible = ref(false)

const userNameInput = ref(store.userName)
watch(() => store.userName, (n) => { userNameInput.value = n }, { immediate: true })

const commitUserName = () => {
  const name = userNameInput.value.trim()
  if (name && name !== store.userName) {
    store.updateUserName(name)
    ElMessage.success(`昵称已更新为: ${name}`)
  } else if (!name) {
    userNameInput.value = store.userName
  }
}

function handleCancel() {
  tool.cancelDraft()
  tool.setTool(null)
}

async function handleDelete(id: string) {
  try {
    await ElMessageBox.confirm('确认删除该标注?', '删除确认', {
      type: 'warning'
    })
    await store.deleteAnnotation(id)
    ElMessage.success('删除成功')
  } catch {
  }
}

async function handleRestore(snapshotId: string) {
  try {
    await ElMessageBox.confirm(
      '回滚到此版本将覆盖当前所有标注，确认继续?',
      '回滚确认',
      { type: 'warning' }
    )
    await store.restoreSnapshot(snapshotId)
    ElMessage.success('回滚成功')
  } catch {
  }
}

function formatTime(timestamp: number) {
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}
</script>

<style lang="scss" scoped>
.annotation-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a2e;
  overflow-y: auto;
}

.panel-header {
  padding: 16px;
  border-bottom: 1px solid #2a2a4a;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;

  h3 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    white-space: nowrap;
  }

  .name-input {
    flex: 1;
    min-width: 0;
    max-width: 140px;

    :deep(.el-input__inner) {
      text-align: right;
      font-size: 12px;
    }
  }
}

.section {
  padding: 16px;
  border-bottom: 1px solid #2a2a4a;
  flex-shrink: 0;
}

.tools-section {
  padding: 16px;
  border-bottom: 1px solid #2a2a4a;
  flex-shrink: 0;
}

.section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  color: #9ca3af;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.tool-buttons {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .el-button {
    width: 100%;
    justify-content: flex-start;
  }
}

.tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 8px 12px;
  background: rgba(64, 158, 255, 0.1);
  border-radius: 6px;
  font-size: 12px;
  color: #60a5fa;
}

.category-group {
  margin-bottom: 12px;

  &:last-child {
    margin-bottom: 0;
  }
}

.group-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
  padding-left: 4px;
}

.category-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.category-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  &.active {
    background: rgba(64, 158, 255, 0.15);
  }

  .color-dot {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    flex-shrink: 0;
    border: 1px solid rgba(255, 255, 255, 0.15);
  }

  .cat-name {
    flex: 1;
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .cat-count {
    font-size: 12px;
    color: #6b7280;
    background: rgba(255, 255, 255, 0.05);
    padding: 2px 8px;
    border-radius: 10px;
  }
}

.annotation-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 200px;
  overflow-y: auto;
}

.annotation-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  &.selected {
    background: rgba(64, 158, 255, 0.15);
  }

  .ann-color {
    width: 4px;
    height: 24px;
    border-radius: 2px;
    flex-shrink: 0;
  }

  .ann-info {
    flex: 1;
    min-width: 0;

    .ann-label {
      font-size: 13px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .ann-meta {
      font-size: 11px;
      color: #6b7280;
    }
  }
}

.snapshot-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 180px;
  overflow-y: auto;
}

.snapshot-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 6px;

  .snap-info {
    flex: 1;
    min-width: 0;

    .snap-message {
      font-size: 12px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .snap-time {
      font-size: 11px;
      color: #6b7280;
    }
  }
}

.empty-annotations {
  text-align: center;
  padding: 16px;
  color: #6b7280;
  font-size: 12px;
}

.shortcuts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.shortcut-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #9ca3af;

  kbd {
    background: #2a2a4a;
    border: 1px solid #3a3a5e;
    border-radius: 4px;
    padding: 2px 6px;
    font-family: monospace;
    font-size: 11px;
    color: #e0e0e0;
  }
}
</style>
