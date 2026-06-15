<template>
  <el-dialog
    v-model="visible"
    title="目标分类模板管理"
    width="560px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="dialog-content">
      <div class="section-header">
        <h4>新增自定义分类</h4>
      </div>

      <el-form :model="newCategory" label-width="80px" size="default" class="new-form">
        <el-form-item label="名称" :error="newCatErrors.name">
          <el-input
            v-model="newCategory.name"
            placeholder="例如: 水雷、桥墩..."
            maxlength="20"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="配色" :error="newCatErrors.color">
          <div class="color-picker-row">
            <el-color-picker
              v-model="newCategory.color"
              :show-alpha="false"
              color-format="hex"
              size="default"
            />
            <el-input
              v-model="newCategory.color"
              placeholder="#RRGGBB"
              class="color-input"
              maxlength="7"
            />
            <div class="preset-colors">
              <span
                v-for="c in presetColors"
                :key="c"
                class="preset-color"
                :style="{ backgroundColor: c }"
                :title="c"
                @click="newCategory.color = c"
              />
            </div>
          </div>
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="newCategory.description"
            type="textarea"
            :rows="2"
            placeholder="分类说明，可留空"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            :icon="Plus"
            :loading="creating"
            @click="handleCreate"
          >
            添加分类
          </el-button>
          <el-button :icon="RefreshRight" @click="resetNewCategory">重置</el-button>
        </el-form-item>
      </el-form>

      <el-divider />

      <div class="section-header">
        <h4>分类模板列表</h4>
        <el-tag size="small" type="info">共 {{ store.categories.length }} 个</el-tag>
      </div>

      <div class="category-list">
        <div
          v-for="cat in store.categories"
          :key="cat.id"
          class="category-row"
        >
          <span class="color-swatch" :style="{ backgroundColor: cat.color }" />

          <div class="cat-info">
            <div class="cat-name">
              {{ cat.name }}
              <el-tag
                v-if="cat.isBuiltin"
                size="small"
                type="info"
                effect="plain"
              >
                内置
              </el-tag>
              <el-tag
                v-else-if="cat.userId"
                size="small"
                type="success"
                effect="plain"
              >
                我的
              </el-tag>
            </div>
            <div v-if="cat.description" class="cat-desc">{{ cat.description }}</div>
          </div>

          <div class="cat-actions" v-if="!cat.isBuiltin && cat.userId">
            <el-button
              :icon="Edit"
              size="small"
              text
              type="primary"
              @click="startEdit(cat)"
            >
              编辑
            </el-button>
            <el-button
              :icon="Delete"
              size="small"
              text
              type="danger"
              @click="handleDelete(cat)"
            >
              删除
            </el-button>
          </div>
          <el-tooltip v-else content="内置分类不可编辑删除" placement="left">
            <el-button
              :icon="Lock"
              size="small"
              text
              :disabled="true"
            />
          </el-tooltip>
        </div>

        <div v-if="store.categories.length === 0" class="empty-list">
          暂无分类
        </div>
      </div>
    </div>

    <el-dialog
      v-model="editDialogVisible"
      title="编辑分类"
      width="420px"
      append-to-body
    >
      <el-form :model="editingCategory" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="editingCategory.name" maxlength="20" show-word-limit />
        </el-form-item>
        <el-form-item label="配色">
          <div class="color-picker-row">
            <el-color-picker v-model="editingCategory.color" :show-alpha="false" color-format="hex" />
            <el-input v-model="editingCategory.color" class="color-input" maxlength="7" />
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="editingCategory.description"
            type="textarea"
            :rows="2"
            maxlength="200"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="updating" @click="handleUpdate">保存</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, RefreshRight, Edit, Delete, Lock } from '@element-plus/icons-vue'
import { useAnnotationStore } from '@/stores/annotation'
import type { Category } from '@/types'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean] }>()

const store = useAnnotationStore()

const visible = ref(props.modelValue)
watch(() => props.modelValue, (v) => { visible.value = v })
watch(visible, (v) => { emit('update:modelValue', v) })

const presetColors = [
  '#ff4d4f', '#faad14', '#f5222d', '#eb2f96',
  '#722ed1', '#2f54eb', '#1890ff', '#13c2c2',
  '#52c41a', '#a0d911', '#fadb14', '#8c8c8c'
]

const newCategory = reactive({
  name: '',
  color: '#1890ff',
  description: ''
})

const newCatErrors = reactive({
  name: '',
  color: ''
})

const creating = ref(false)
const updating = ref(false)

const editDialogVisible = ref(false)
const editingCategory = reactive<{
  id: string
  name: string
  color: string
  description: string
}>({ id: '', name: '', color: '#1890ff', description: '' })

const resetNewCategory = () => {
  newCategory.name = ''
  newCategory.color = '#1890ff'
  newCategory.description = ''
  newCatErrors.name = ''
  newCatErrors.color = ''
}

const validateHexColor = (s: string): boolean => {
  return /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/.test(s)
}

const handleCreate = async () => {
  newCatErrors.name = ''
  newCatErrors.color = ''

  if (!newCategory.name.trim()) {
    newCatErrors.name = '请输入分类名称'
    return
  }
  if (!validateHexColor(newCategory.color)) {
    newCatErrors.color = '请输入 #RRGGBB 格式'
    return
  }

  try {
    creating.value = true
    await store.createCategory({
      name: newCategory.name.trim(),
      color: newCategory.color,
      description: newCategory.description.trim() || undefined
    })
    ElMessage.success('分类创建成功')
    resetNewCategory()
  } catch (e: any) {
    const msg = e?.response?.data?.error || '创建失败'
    ElMessage.error(msg)
  } finally {
    creating.value = false
  }
}

const startEdit = (cat: Category) => {
  editingCategory.id = cat.id
  editingCategory.name = cat.name
  editingCategory.color = cat.color
  editingCategory.description = cat.description
  editDialogVisible.value = true
}

const handleUpdate = async () => {
  if (!editingCategory.name.trim()) {
    ElMessage.warning('请输入分类名称')
    return
  }
  if (!validateHexColor(editingCategory.color)) {
    ElMessage.warning('请输入 #RRGGBB 格式的颜色')
    return
  }
  try {
    updating.value = true
    await store.updateCategory(editingCategory.id, {
      name: editingCategory.name.trim(),
      color: editingCategory.color,
      description: editingCategory.description.trim() || undefined
    })
    ElMessage.success('分类已更新')
    editDialogVisible.value = false
  } catch (e: any) {
    const msg = e?.response?.data?.error || '更新失败'
    ElMessage.error(msg)
  } finally {
    updating.value = false
  }
}

const handleDelete = async (cat: Category) => {
  const usingCount = store.annotations.filter(a => a.categoryId === cat.id).length
  const tip = usingCount > 0
    ? `该分类下存在 ${usingCount} 个标注，删除后标注的关联分类会被清空（但标注本身保留），确认删除?`
    : `确认删除分类 "${cat.name}"?`

  try {
    await ElMessageBox.confirm(tip, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
    await store.deleteCategory(cat.id)
    ElMessage.success('分类已删除')
  } catch {
  }
}

const handleClose = () => {
  resetNewCategory()
  editDialogVisible.value = false
}
</script>

<style lang="scss" scoped>
.dialog-content {
  max-height: 60vh;
  overflow-y: auto;
  padding: 0 8px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;

  h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
  }
}

.new-form {
  margin-bottom: 8px;
}

.color-picker-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;

  .color-input {
    width: 130px;
  }
}

.preset-colors {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.preset-color {
  width: 22px;
  height: 22px;
  border-radius: 4px;
  cursor: pointer;
  border: 2px solid rgba(255, 255, 255, 0.1);
  transition: transform 0.15s;

  &:hover {
    transform: scale(1.15);
  }
}

.category-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.category-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid #2a2a4a;
  border-radius: 8px;
  transition: background 0.2s;

  &:hover {
    background: rgba(255, 255, 255, 0.06);
  }
}

.color-swatch {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  flex-shrink: 0;
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.cat-info {
  flex: 1;
  min-width: 0;

  .cat-name {
    font-size: 13px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .cat-desc {
    font-size: 11px;
    color: #6b7280;
    margin-top: 2px;
  }
}

.cat-actions {
  display: flex;
  gap: 4px;
}

.empty-list {
  text-align: center;
  padding: 24px;
  color: #6b7280;
  font-size: 13px;
}
</style>
