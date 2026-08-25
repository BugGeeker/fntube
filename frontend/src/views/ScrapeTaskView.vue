<template>
  <div>
    <a-card title="刮削计划任务">
      <template #extra>
        <a-button type="primary" @click="openCreate">
          <template #icon>
            <PlusOutlined />
          </template>
          新建任务
        </a-button>
      </template>
      <a-spin :spinning="loading">
        <a-table :dataSource="tasks" :columns="columns" rowKey="id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'enabled'">
              <a-tag v-if="record.enabled" color="green">启用</a-tag>
              <a-tag v-else color="default">停用</a-tag>
            </template>
            <template v-if="column.key === 'interval'">
              {{ record.interval }} 分钟
            </template>
            <template v-if="column.key === 'last_run_at'">
              {{ record.last_run_at ? formatDateTime(record.last_run_at) : '未执行' }}
            </template>
            <template v-if="column.key === 'action'">
              <a-space>
                <a-button size="small" @click="handleRun(record)" :loading="runningId === record.id">执行</a-button>
                <a-button size="small" @click="openEdit(record)">编辑</a-button>
                <a-popconfirm title="确认删除？" @confirm="handleDelete(record)">
                  <a-button danger size="small">删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-spin>
    </a-card>

    <!-- 新建/编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingTask ? '编辑任务' : '新建任务'" @ok="handleSave" :confirm-loading="saving">
      <a-form layout="vertical">
        <a-form-item label="任务名称" required>
          <a-input v-model:value="form.name" placeholder="请输入任务名称" />
        </a-form-item>
        <a-form-item label="媒体库" required>
          <a-select v-model:value="form.library_id" placeholder="选择媒体库" :loading="libsLoading" @change="onLibraryChange">
            <a-select-option v-for="lib in libraries" :key="lib.id" :value="lib.id">{{ lib.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="扫描频率（分钟）" required>
          <a-input-number v-model:value="form.interval" :min="1" :max="10080" style="width: 100%" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getScrapeTasks, createScrapeTask, updateScrapeTask, deleteScrapeTask, runScrapeTask, type ScrapeTask } from '@/api/scrapetask'
import { getLibraries, type Library } from '@/api/trimmedia'

const loading = ref(false)
const saving = ref(false)
const tasks = ref<ScrapeTask[]>([])
const runningId = ref<number | null>(null)
const modalVisible = ref(false)
const editingTask = ref<ScrapeTask | null>(null)
const libraries = ref<Library[]>([])
const libsLoading = ref(false)

const form = ref({
  name: '',
  library_id: '',
  library_name: '',
  interval: 60,
  enabled: true,
})

const columns = [
  { title: '任务名称', dataIndex: 'name', key: 'name' },
  { title: '媒体库', dataIndex: 'library_name', key: 'library_name' },
  { title: '扫描频率', key: 'interval', width: 120 },
  { title: '状态', key: 'enabled', width: 80 },
  { title: '上次执行', key: 'last_run_at', width: 180 },
  { title: '操作', key: 'action', width: 200 },
]

function formatDateTime(date: string): string {
  if (!date) return ''
  const d = new Date(date)
  if (isNaN(d.getTime())) return date
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${m}-${day} ${h}:${min}`
}

async function loadTasks() {
  loading.value = true
  try {
    const { data } = await getScrapeTasks()
    tasks.value = data || []
  } catch {
    message.error('获取任务列表失败')
  } finally {
    loading.value = false
  }
}

async function loadLibraries() {
  libsLoading.value = true
  try {
    const { data } = await getLibraries()
    libraries.value = data || []
  } catch {
    // 静默失败
  } finally {
    libsLoading.value = false
  }
}

function openCreate() {
  editingTask.value = null
  form.value = { name: '', library_id: '', library_name: '', interval: 60, enabled: true }
  modalVisible.value = true
}

function openEdit(task: ScrapeTask) {
  editingTask.value = task
  form.value = {
    name: task.name,
    library_id: task.library_id,
    library_name: task.library_name,
    interval: task.interval,
    enabled: task.enabled,
  }
  modalVisible.value = true
}

function onLibraryChange(value: string) {
  const lib = libraries.value.find(l => l.id === value)
  if (lib) {
    form.value.library_name = lib.name
  }
}

async function handleSave() {
  if (!form.value.name) {
    message.warning('请输入任务名称')
    return
  }
  if (!form.value.library_id) {
    message.warning('请选择媒体库')
    return
  }
  saving.value = true
  try {
    if (editingTask.value) {
      await updateScrapeTask({ id: editingTask.value.id, ...form.value })
      message.success('更新成功')
    } else {
      await createScrapeTask(form.value)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadTasks()
  } catch {
    message.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(task: ScrapeTask) {
  try {
    await deleteScrapeTask(task.id)
    message.success('删除成功')
    loadTasks()
  } catch {
    message.error('删除失败')
  }
}

async function handleRun(task: ScrapeTask) {
  runningId.value = task.id
  try {
    const { data } = await runScrapeTask(task.id)
    message.success(data.message || '任务已开始执行')
    // 延迟刷新以获取最新状态
    setTimeout(() => loadTasks(), 3000)
  } catch {
    message.error('执行失败')
  } finally {
    runningId.value = null
  }
}

onMounted(() => {
  loadTasks()
  loadLibraries()
})
</script>
