<template>
  <div>
    <a-card title="刮削计划任务">
      <template #extra>
        <a-space>
          <a-button @click="showRunRecords = true">
            <template #icon>
              <HistoryOutlined />
            </template>
            运行记录
          </a-button>
          <a-button type="primary" @click="openCreate">
            <template #icon>
              <PlusOutlined />
            </template>
            新建任务
          </a-button>
        </a-space>
      </template>
      <a-spin :spinning="loading">
        <a-table :dataSource="tasks" :columns="columns" rowKey="id" :pagination="false" :scroll="{ x: '100%' }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'enabled'">
              <a-tag v-if="record.enabled" color="green">启用</a-tag>
              <a-tag v-else color="default">停用</a-tag>
            </template>
            <template v-if="column.key === 'run_status'">
              <a-tag v-if="record.is_running" color="processing">运行中</a-tag>
              <a-tag v-else color="default">空闲中</a-tag>
            </template>
            <template v-if="column.key === 'interval'">
              {{ record.interval }} 分钟
            </template>
            <template v-if="column.key === 'last_run_at'">
              {{ record.last_run_at ? formatDateTime(record.last_run_at) : '-' }}
            </template>
            <template v-if="column.key === 'action'">
              <a-space>
                <a-button size="small" @click="handleRun(record)" :loading="runningId === record.id" :disabled="record.is_running">执行</a-button>
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

    <!-- 运行记录抽屉 -->
    <a-drawer
      v-model:open="showRunRecords"
      title="运行记录"
      placement="right"
      width="700"
    >
      <a-spin :spinning="runRecordLoading">
        <a-table
          :dataSource="runRecords"
          :columns="runRecordColumns"
          rowKey="id"
          :pagination="runRecordPagination"
          @change="handleRunRecordTableChange"
          size="small"
          :scroll="{ x: '100%' }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'start_time'">
              {{ formatDateTime(record.start_time) }}
            </template>
            <template v-if="column.key === 'duration'">
              {{ formatDuration(record.duration) }}
            </template>
            <template v-if="column.key === 'counts'">
              <a-space size="small">
                <a-tag color="green" style="cursor: pointer" @click="openDetail(record.id, 'completed')">{{ record.completed_count }}完成</a-tag>
                <a-tag color="blue" style="cursor: pointer" @click="openDetail(record.id, 'success')">{{ record.success_count }}成功</a-tag>
                <a-tag v-if="record.failed_count > 0" color="red" style="cursor: pointer" @click="openDetail(record.id, 'failed')">{{ record.failed_count }}失败</a-tag>
              </a-space>
            </template>
            <template v-if="column.key === 'status'">
              <a-tag v-if="record.status === 'running'" color="processing">运行中</a-tag>
              <a-tag v-else-if="record.status === 'done'" color="green">完成</a-tag>
              <a-tag v-else-if="record.status === 'error'" color="red">错误</a-tag>
              <a-tag v-else color="default">{{ record.status }}</a-tag>
            </template>
          </template>
        </a-table>
      </a-spin>
    </a-drawer>

    <!-- 运行记录明细抽屉 -->
    <a-drawer
      v-model:open="showDetail"
      title="运行明细"
      placement="right"
      width="800"
    >
      <a-spin :spinning="detailLoading">
        <div v-if="detailRecord" style="margin-bottom: 16px">
          <a-descriptions :column="2" size="small" bordered>
            <a-descriptions-item label="任务名称">{{ detailRecord.task_name }}</a-descriptions-item>
            <a-descriptions-item label="媒体库">{{ detailRecord.library_name }}</a-descriptions-item>
            <a-descriptions-item label="开始时间">{{ formatDateTime(detailRecord.start_time) }}</a-descriptions-item>
            <a-descriptions-item label="运行时长">{{ formatDuration(detailRecord.duration) }}</a-descriptions-item>
            <a-descriptions-item label="成功">
              <a-tag color="blue">{{ detailRecord.success_count }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="完成">
              <a-tag color="green">{{ detailRecord.completed_count }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="失败">
              <a-tag color="red">{{ detailRecord.failed_count }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="状态">
              <a-tag v-if="detailRecord.status === 'running'" color="processing">运行中</a-tag>
              <a-tag v-else-if="detailRecord.status === 'done'" color="green">完成</a-tag>
              <a-tag v-else-if="detailRecord.status === 'error'" color="red">错误</a-tag>
            </a-descriptions-item>
            <a-descriptions-item v-if="detailRecord.error" label="错误" :span="2">
              <span style="color: #ff4d4f">{{ detailRecord.error }}</span>
            </a-descriptions-item>
          </a-descriptions>
        </div>
        <a-table
          :dataSource="detailLogs"
          :columns="detailColumns"
          rowKey="id"
          :pagination="{ pageSize: 50, hideOnSinglePage: true }"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag v-if="record.status === 'in_progress'" color="processing">刮削中</a-tag>
              <a-tag v-else-if="record.status === 'success'" color="green">成功</a-tag>
              <a-tag v-else-if="record.status === 'failed'" color="red">失败</a-tag>
              <a-tag v-else-if="record.status === 'completed'" color="blue">完成</a-tag>
              <a-tag v-else color="default">{{ record.status }}</a-tag>
            </template>
            <template v-if="column.key === 'error'">
              <a-tooltip v-if="record.error" :title="record.error">
                <span style="color: #ff4d4f; cursor: pointer">{{ truncate(record.error, 30) }}</span>
              </a-tooltip>
              <span v-else>-</span>
            </template>
            <template v-if="column.key === 'created_at'">
              {{ formatDate(record.created_at) }}
            </template>
          </template>
        </a-table>
      </a-spin>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, HistoryOutlined } from '@ant-design/icons-vue'
import { getScrapeTasks, createScrapeTask, updateScrapeTask, deleteScrapeTask, runScrapeTask, type ScrapeTask } from '@/api/scrapetask'
import { getLibraries, type Library } from '@/api/trimmedia'
import { getTaskRunRecords, getTaskRunDetail, type TaskRunRecord } from '@/api/taskrun'
import type { ScrapeLog } from '@/api/scrapelog'
import { formatDateTime, formatDate } from '@/utils/format'

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
  { title: '任务名称', dataIndex: 'name', key: 'name', fixed: 'left', width: 120 },
  { title: '媒体库', dataIndex: 'library_name', key: 'library_name', width: 120 },
  { title: '扫描频率', key: 'interval', width: 120 },
  { title: '状态', key: 'enabled', width: 80 },
  { title: '运行状态', key: 'run_status', width: 100 },
  { title: '上次执行', key: 'last_run_at', width: 180 },
  { title: '操作', key: 'action', fixed: 'right', width: 200 },
]

// 运行记录抽屉
const showRunRecords = ref(false)
const runRecordLoading = ref(false)
const runRecords = ref<TaskRunRecord[]>([])
const runRecordCurrentPage = ref(1)
const runRecordPageSize = ref(20)
const runRecordPagination = ref({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})

const runRecordColumns = [
  { title: '任务名称', dataIndex: 'task_name', key: 'task_name', ellipsis: true, width: 120 },
  { title: '媒体库', dataIndex: 'library_name', key: 'library_name', ellipsis: true, width: 120 },
  { title: '开始时间', key: 'start_time', width: 150 },
  { title: '数量', key: 'counts', width: 180 },
  { title: '运行时长', key: 'duration', width: 100 },
  { title: '状态', key: 'status', width: 80 },
]

// 明细抽屉
const showDetail = ref(false)
const detailLoading = ref(false)
const detailRecord = ref<TaskRunRecord | null>(null)
const detailLogs = ref<ScrapeLog[]>([])

const detailColumns = [
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '番号', dataIndex: 'number', key: 'number', width: 120 },
  { title: '状态', key: 'status', width: 100 },
  { title: '错误信息', key: 'error', width: 200, ellipsis: true },
  { title: '刮削时间', key: 'created_at', width: 160 },
]

function formatDuration(seconds: number): string {
  if (!seconds || seconds <= 0) return '-'
  if (seconds < 60) return `${seconds}秒`
  const min = Math.floor(seconds / 60)
  const sec = seconds % 60
  if (min < 60) return `${min}分${sec}秒`
  const h = Math.floor(min / 60)
  const remainMin = min % 60
  return `${h}小时${remainMin}分`
}

function truncate(s: string, n: number): string {
  if (!s) return ''
  return s.length > n ? s.slice(0, n) + '…' : s
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
  if (task.is_running) {
    message.warning('任务正在运行中，请等待完成')
    return
  }
  runningId.value = task.id
  try {
    const { data } = await runScrapeTask(task.id)
    message.success(data.message || '任务已开始执行')
    // 延迟刷新以获取最新状态
    setTimeout(() => loadTasks(), 3000)
  } catch (err: any) {
    const status = err?.response?.status
    if (status === 409) {
      message.warning('任务正在运行中，请等待完成')
      loadTasks()
    } else {
      message.error('执行失败')
    }
  } finally {
    runningId.value = null
  }
}

// 运行记录
async function loadRunRecords() {
  runRecordLoading.value = true
  try {
    const { data } = await getTaskRunRecords(runRecordCurrentPage.value, runRecordPageSize.value)
    runRecords.value = data.items || []
    runRecordPagination.value.current = runRecordCurrentPage.value
    runRecordPagination.value.pageSize = runRecordPageSize.value
    runRecordPagination.value.total = data.total
  } catch {
    message.error('获取运行记录失败')
  } finally {
    runRecordLoading.value = false
  }
}

function handleRunRecordTableChange(pag: { current: number; pageSize: number }) {
  runRecordCurrentPage.value = pag.current
  runRecordPageSize.value = pag.pageSize
  loadRunRecords()
}

// 明细
async function openDetail(id: number, _filter?: string) {
  showDetail.value = true
  detailLoading.value = true
  try {
    const { data } = await getTaskRunDetail(id)
    detailRecord.value = data.record
    detailLogs.value = data.logs || []
  } catch {
    message.error('获取明细失败')
  } finally {
    detailLoading.value = false
  }
}

onMounted(() => {
  loadTasks()
  loadLibraries()
})

watch(showRunRecords, (val) => {
  if (val) {
    loadRunRecords()
  }
})
</script>
