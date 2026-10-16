<template>
  <div class="storage-config-page">
    <a-card title="存储设置">
      <template #extra>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新增映射
        </a-button>
      </template>

      <a-spin :spinning="loading">
        <a-empty v-if="!loading && mappings.length === 0" description="暂无存储映射" />

        <div v-else class="desktop-table">
          <a-table
            :data-source="mappings"
            :columns="columns"
            row-key="id"
            :pagination="false"
            :scroll="{ x: 900 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button size="small" @click="openEdit(record)">编辑</a-button>
                  <a-popconfirm title="确认删除这条映射？" @confirm="removeMapping(record)">
                    <a-button danger size="small">删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <div v-if="mappings.length" class="mobile-list">
          <div v-for="mapping in mappings" :key="mapping.id" class="mapping-item">
            <div class="mapping-title">{{ mapping.name }}</div>
            <div class="mapping-row">
              <span>光鸭云盘目录</span>
              <span>{{ mapping.guangya_folder_path }}</span>
            </div>
            <div class="mapping-row">
              <span>SmartStrm任务</span>
              <span>{{ mapping.smartstrm_task_name }}</span>
            </div>
            <div class="mapping-row">
              <span>Strm保存目录</span>
              <span>{{ mapping.strm_save_path }}</span>
            </div>
            <a-space class="mapping-actions">
              <a-button size="small" @click="openEdit(mapping)">编辑</a-button>
              <a-popconfirm title="确认删除这条映射？" @confirm="removeMapping(mapping)">
                <a-button danger size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </div>
        </div>
      </a-spin>
    </a-card>

    <a-modal
      v-model:open="editorOpen"
      :title="editingId ? '编辑存储映射' : '新增存储映射'"
      ok-text="保存"
      :confirm-loading="saving"
      @ok="saveMapping"
      @cancel="closeEditor"
    >
      <a-form layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="form.name" placeholder="请输入映射名称" allow-clear />
        </a-form-item>
        <a-form-item label="光鸭云盘目录" required>
          <a-input-group compact class="folder-input">
            <a-input
              :value="form.guangya_folder_path"
              placeholder="请选择光鸭云盘目录"
              readonly
            />
            <a-button @click="openFolderPicker">
              <template #icon><FolderOpenOutlined /></template>
              选择
            </a-button>
          </a-input-group>
        </a-form-item>
        <a-form-item label="SmartStrm任务" required>
          <a-select
            v-model:value="form.smartstrm_task_name"
            placeholder="请选择SmartStrm任务"
            :options="smartStrmTaskOptions"
            :loading="loadingSmartStrmTasks"
            :not-found-content="loadingSmartStrmTasks ? undefined : '暂无可用任务'"
            show-search
            option-filter-prop="label"
          />
        </a-form-item>
        <a-form-item label="Strm保存目录" required>
          <a-input
            v-model:value="form.strm_save_path"
            placeholder="请输入Strm保存目录"
            allow-clear
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <GuangyaFolderPickerDialog
      v-model:open="folderOpen"
      :selected-id="form.guangya_folder_id"
      :selected-path="form.guangya_folder_path"
      @select="selectFolder"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import { FolderOpenOutlined, PlusOutlined } from '@ant-design/icons-vue'
import GuangyaFolderPickerDialog, {
  type GuangyaFolderSelection,
} from '@/components/GuangyaFolderPickerDialog.vue'
import {
  createStorageMapping,
  deleteStorageMapping,
  getStorageMappings,
  updateStorageMapping,
  type StorageMapping,
  type StorageMappingInput,
} from '@/api/storageconfig'
import { getSmartStrmTasks } from '@/api/smartstrm'

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
  { title: '光鸭云盘目录', dataIndex: 'guangya_folder_path', key: 'guangya_folder_path', ellipsis: true, width: 220 },
  { title: 'SmartStrm 任务名', dataIndex: 'smartstrm_task_name', key: 'smartstrm_task_name', ellipsis: true, width: 200 },
  { title: 'strm 保存目录', dataIndex: 'strm_save_path', key: 'strm_save_path', ellipsis: true, width: 220 },
  { title: '操作', key: 'action', fixed: 'right', width: 140 },
]

const mappings = ref<StorageMapping[]>([])
const loading = ref(false)
const saving = ref(false)
const loadingSmartStrmTasks = ref(false)
const smartStrmTaskOptions = ref<{ label: string; value: string }[]>([])
const editorOpen = ref(false)
const editingId = ref(0)
const form = reactive<StorageMappingInput>({
  name: '',
  guangya_folder_id: '',
  guangya_folder_path: '',
  smartstrm_task_name: '',
  strm_save_path: '',
})

const folderOpen = ref(false)

function errorMessage(error: unknown, fallback: string): string {
  const candidate = error as { message?: string; response?: { data?: { error?: string } } }
  return candidate.response?.data?.error || candidate.message || fallback
}

async function loadMappings() {
  loading.value = true
  try {
    const { data } = await getStorageMappings()
    mappings.value = data || []
  } catch (error) {
    message.error(errorMessage(error, '读取存储配置失败'))
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = 0
  Object.assign(form, {
    name: '',
    guangya_folder_id: '',
    guangya_folder_path: '',
    smartstrm_task_name: '',
    strm_save_path: '',
  })
}

async function loadSmartStrmTasks() {
  loadingSmartStrmTasks.value = true
  try {
    const { data } = await getSmartStrmTasks()
    smartStrmTaskOptions.value = Object.entries(data)
      .map(([key, task]) => {
        const name = task.name?.trim() || key
        return { label: name, value: name }
      })
      .sort((left, right) => left.label.localeCompare(right.label, 'zh-CN'))
  } catch (error) {
    smartStrmTaskOptions.value = []
    message.error(errorMessage(error, '读取 SmartStrm 任务失败'))
  } finally {
    loadingSmartStrmTasks.value = false
  }
}

function ensureCurrentTaskOption(taskName: string) {
  if (!taskName || smartStrmTaskOptions.value.some((option) => option.value === taskName)) return
  smartStrmTaskOptions.value.push({ label: taskName, value: taskName })
}

async function openCreate() {
  resetForm()
  editorOpen.value = true
  await loadSmartStrmTasks()
}

async function openEdit(mapping: StorageMapping) {
  editingId.value = mapping.id
  Object.assign(form, {
    name: mapping.name,
    guangya_folder_id: mapping.guangya_folder_id,
    guangya_folder_path: mapping.guangya_folder_path,
    smartstrm_task_name: mapping.smartstrm_task_name,
    strm_save_path: mapping.strm_save_path,
  })
  editorOpen.value = true
  await loadSmartStrmTasks()
  ensureCurrentTaskOption(mapping.smartstrm_task_name)
}

function closeEditor() {
  editorOpen.value = false
  resetForm()
}

async function saveMapping() {
  const payload: StorageMappingInput = {
    name: form.name.trim(),
    guangya_folder_id: form.guangya_folder_id,
    guangya_folder_path: form.guangya_folder_path.trim(),
    smartstrm_task_name: form.smartstrm_task_name.trim(),
    strm_save_path: form.strm_save_path.trim(),
  }
  if (!payload.name || !payload.guangya_folder_path || !payload.smartstrm_task_name || !payload.strm_save_path) {
    message.warning('请填写完整的存储映射')
    return
  }

  saving.value = true
  try {
    if (editingId.value) {
      await updateStorageMapping(editingId.value, payload)
      message.success('存储映射已更新')
    } else {
      await createStorageMapping(payload)
      message.success('存储映射已创建')
    }
    closeEditor()
    await loadMappings()
  } catch (error) {
    message.error(errorMessage(error, '保存存储映射失败'))
  } finally {
    saving.value = false
  }
}

async function removeMapping(mapping: StorageMapping) {
  try {
    await deleteStorageMapping(mapping.id)
    message.success('存储映射已删除')
    await loadMappings()
  } catch (error) {
    message.error(errorMessage(error, '删除存储映射失败'))
  }
}

function openFolderPicker() {
  folderOpen.value = true
}

function selectFolder(selection: GuangyaFolderSelection) {
  form.guangya_folder_id = selection.id
  form.guangya_folder_path = selection.path
}

onMounted(loadMappings)
</script>

<style scoped lang="scss">
.mobile-list {
  display: none;
}

.folder-input {
  display: flex;

  .ant-input {
    flex: 1;
    min-width: 0;
  }
}

.mapping-item {
  padding: 12px 0;
  border-bottom: 1px solid rgba(5, 5, 5, 0.08);

  &:last-child {
    border-bottom: 0;
  }
}

.mapping-title {
  margin-bottom: 10px;
  font-weight: 600;
}

.mapping-row {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  gap: 12px;
  margin: 8px 0;

  span:last-child {
    overflow-wrap: anywhere;
    text-align: right;
  }
}

.mapping-actions {
  margin-top: 10px;
}

@media (max-width: 639px) {
  .desktop-table {
    display: none;
  }

  .mobile-list {
    display: block;
  }
}
</style>
