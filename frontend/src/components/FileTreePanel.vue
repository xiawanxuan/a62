<template>
  <div class="file-tree-panel">
    <div class="panel-header">
      <h3><el-icon><FolderOpened /></el-icon> 声呐文件</h3>
      <el-button :icon="Upload" size="small" @click="showUpload = true">上传</el-button>
    </div>

    <div class="search-box">
      <el-input
        v-model="searchQuery"
        :prefix-icon="Search"
        placeholder="搜索文件..."
        clearable
      />
    </div>

    <div class="tree-container">
      <el-tree
        :data="treeData"
        :props="treeProps"
        node-key="id"
        :highlight-current="true"
        :expand-on-click-node="false"
        :default-expand-all="true"
        @node-click="handleNodeClick"
        :empty-text="store.isLoading ? '加载中...' : '暂无文件'"
      >
        <template #default="{ node, data }">
          <div class="tree-node">
            <span class="node-icon">
              <el-icon v-if="data.type === 'folder'"><Folder /></el-icon>
              <el-icon v-else><Picture /></el-icon>
            </span>
            <span class="node-label" :title="data.name">{{ data.name }}</span>
            <span v-if="data.fileInfo" class="node-count">
              {{ data.fileInfo.annotationCount }}
            </span>
          </div>
        </template>
      </el-tree>
    </div>

    <el-dialog v-model="showUpload" title="上传声呐文件" width="480px">
      <el-upload
        drag
        :action="uploadAction"
        :headers="uploadHeaders"
        :show-file-list="false"
        :before-upload="beforeUpload"
        @success="handleUploadSuccess"
        @error="handleUploadError"
        accept="image/*"
      >
        <el-icon class="upload-icon"><UploadFilled /></el-icon>
        <div class="upload-text">拖拽声呐图像到这里，或<em>点击选择</em></div>
        <div class="upload-tip">支持 PNG、JPG、BMP 格式，最大 100MB</div>
      </el-upload>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import {
  FolderOpened, Upload, Search, Folder, Picture, UploadFilled
} from '@element-plus/icons-vue'
import { useAnnotationStore } from '@/stores/annotation'
import type { FileNode } from '@/types'

const emit = defineEmits<{
  selectFile: [fileId: string]
}>()

const store = useAnnotationStore()
const showUpload = ref(false)
const searchQuery = ref('')

const uploadAction = '/api/files'
const uploadHeaders = {}

const treeData = computed<FileNode[]>(() => {
  let files = store.files
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    files = files.filter(f => f.name.toLowerCase().includes(q))
  }

  return files.map(f => ({
    id: f.id,
    name: f.name,
    type: 'file' as const,
    fileInfo: f
  }))
})

const treeProps = {
  label: 'name',
  children: 'children',
  isLeaf: (data: FileNode) => data.type === 'file'
}

function handleNodeClick(data: FileNode) {
  if (data.type === 'file' && data.fileInfo) {
    emit('selectFile', data.fileInfo.id)
  }
}

function beforeUpload(file: File) {
  const maxSize = 100 * 1024 * 1024
  if (file.size > maxSize) {
    ElMessage.error('文件大小不能超过 100MB')
    return false
  }

  const validTypes = ['image/png', 'image/jpeg', 'image/jpg', 'image/bmp']
  if (!validTypes.includes(file.type)) {
    ElMessage.error('只支持 PNG、JPG、BMP 格式')
    return false
  }

  return true
}

async function handleUploadSuccess() {
  showUpload.value = false
  ElMessage.success('上传成功')
  await store.loadFiles()
}

function handleUploadError() {
  ElMessage.error('上传失败')
}
</script>

<style lang="scss" scoped>
.file-tree-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a2e;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #2a2a4a;

  h3 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    font-size: 15px;
    font-weight: 600;
  }
}

.search-box {
  padding: 12px 16px;
  border-bottom: 1px solid #2a2a4a;
}

.tree-container {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 0;

  .node-icon {
    display: flex;
    align-items: center;
    color: #409eff;
  }

  .node-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
  }

  .node-count {
    background: rgba(64, 158, 255, 0.2);
    color: #409eff;
    padding: 2px 8px;
    border-radius: 10px;
    font-size: 11px;
    font-weight: 500;
  }
}

.upload-icon {
  font-size: 48px;
  color: #409eff;
  margin-bottom: 16px;
}

.upload-text {
  font-size: 14px;
  color: #d1d5db;
  margin-bottom: 8px;

  em {
    color: #409eff;
    font-style: normal;
  }
}

.upload-tip {
  font-size: 12px;
  color: #6b7280;
}
</style>
