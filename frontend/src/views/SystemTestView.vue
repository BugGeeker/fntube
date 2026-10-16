<template>
  <div class="system-test-view">
    <a-card title="文件复制测试">
      <a-alert
        message="测试会实际复制文件或目录；目标路径已存在时不会覆盖。"
        type="warning"
        show-icon
        class="section-alert"
      />
      <a-form layout="vertical">
        <a-form-item label="源目录或文件路径">
          <a-input v-model:value="copyForm.source" placeholder="/vol1/1000/source/file.strm" allow-clear />
        </a-form-item>
        <a-form-item label="目标目录或文件路径">
          <a-input v-model:value="copyForm.destination" placeholder="/vol2/1000/destination/file.strm" allow-clear />
        </a-form-item>
        <a-button type="primary" :loading="copying" @click="runCopyTest">测试复制</a-button>
      </a-form>
      <a-descriptions v-if="copyResult" :column="1" bordered size="small" class="result-block">
        <a-descriptions-item label="实际目标">{{ copyResult.destination }}</a-descriptions-item>
        <a-descriptions-item label="复制结果">
          {{ copyResult.files }} 个文件，{{ copyResult.directories }} 个目录，{{ formatBytes(copyResult.bytes) }}
        </a-descriptions-item>
      </a-descriptions>
    </a-card>

    <a-card title="应用共享授权测试">
      <a-alert
        message="前端授权仅管理员可操作；后端查询需要在飞牛环境中运行。"
        type="info"
        show-icon
        class="section-alert"
      />
      <a-space wrap>
        <a-button type="primary" :loading="authorizing" @click="pickSharedFolder">前端 SDK 选择并授权目录</a-button>
        <a-button :loading="loadingSharedFolders" @click="loadSharedFolders">后端 API 查询授权目录</a-button>
      </a-space>
      <a-input-search
        v-model:value="knownSharedPath"
        placeholder="/vol1/1000/data/shared"
        enter-button="授权指定目录"
        :loading="authorizingKnownPath"
        class="known-path-input"
        @search="authorizeKnownPath"
      />
      <a-list
        v-if="sharedFolders.length > 0"
        :data-source="sharedFolders"
        bordered
        size="small"
        class="result-block"
      >
        <template #renderItem="{ item }">
          <a-list-item>{{ item }}</a-list-item>
        </template>
      </a-list>
      <a-empty v-else-if="sharedFoldersLoaded" description="暂无已授权目录" class="result-block" />
    </a-card>

    <a-card title="文件权限检查">
      <a-form layout="vertical">
        <a-form-item label="用户 UID">
          <a-input-number v-model:value="aclForm.uid" :min="1" :precision="0" class="uid-input" />
        </a-form-item>
        <a-form-item label="文件或目录路径">
          <a-textarea
            v-model:value="aclForm.paths"
            :rows="4"
            placeholder="每行填写一个路径，例如：&#10;/vol1/1000/data/test.txt&#10;/vol1/1000/data/demo"
          />
        </a-form-item>
        <a-button type="primary" :loading="checkingACL" @click="runACLCheck">检查权限</a-button>
      </a-form>
      <a-table
        v-if="aclResults.length > 0"
        :columns="aclColumns"
        :data-source="aclResults"
        :pagination="false"
        row-key="path"
        size="small"
        class="result-block"
        :scroll="{ x: 680 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="['readable', 'writable', 'deletable'].includes(column.dataIndex)">
            <a-tag :color="record[column.dataIndex] ? 'success' : 'default'">
              {{ record[column.dataIndex] ? '是' : '否' }}
            </a-tag>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { TrimApp } from '@trimjs/web-app'
import { message } from 'ant-design-vue'
import axios from 'axios'
import {
  checkACL,
  getSharedFolders,
  testCopy,
  type ACLResult,
  type CopyTestResult,
} from '@/api/systemtest'

const sdk = new TrimApp()
const copyForm = reactive({ source: '', destination: '' })
const copyResult = ref<CopyTestResult>()
const copying = ref(false)
const authorizing = ref(false)
const authorizingKnownPath = ref(false)
const knownSharedPath = ref('')
const loadingSharedFolders = ref(false)
const sharedFoldersLoaded = ref(false)
const sharedFolders = ref<string[]>([])
const aclForm = reactive({ uid: 1000, paths: '' })
const checkingACL = ref(false)
const aclResults = ref<ACLResult[]>([])
const pendingAuthState = ref('')

const aclColumns = [
  { title: '路径', dataIndex: 'path', key: 'path' },
  { title: '可读', dataIndex: 'readable', key: 'readable', width: 90 },
  { title: '可写', dataIndex: 'writable', key: 'writable', width: 90 },
  { title: '可删除', dataIndex: 'deletable', key: 'deletable', width: 90 },
]

function errorMessage(error: unknown, fallback: string) {
  if (axios.isAxiosError(error)) {
    return error.response?.data?.error || error.message || fallback
  }
  return error instanceof Error ? error.message : fallback
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / 1024 ** unitIndex
  return `${value.toFixed(unitIndex === 0 ? 0 : 2)} ${units[unitIndex]}`
}

async function runCopyTest() {
  if (!copyForm.source.trim() || !copyForm.destination.trim()) {
    message.warning('请填写源路径和目标路径')
    return
  }
  copying.value = true
  copyResult.value = undefined
  try {
    const { data } = await testCopy(copyForm.source, copyForm.destination)
    copyResult.value = data
    message.success('文件复制测试成功')
  } catch (error) {
    message.error(errorMessage(error, '文件复制测试失败'))
  } finally {
    copying.value = false
  }
}

function callbackPath() {
  const pathname = window.location.pathname
  const base = pathname.endsWith('/index.html')
    ? pathname.slice(0, -'index.html'.length)
    : pathname.endsWith('/')
      ? pathname
      : `${pathname}/`
  return `${base}app-auth-callback.html`
}

function createAuthState() {
  return typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

async function pickSharedFolder() {
  authorizing.value = true
  try {
    await sdk.ready()
    if (sdk.isStandaloneWeb) {
      const state = createAuthState()
      pendingAuthState.value = state
      await sdk.openAppAuth(
        'pickSharedFile',
        {
          appName: 'FnTube',
          sidebarGroup: ['myFiles', 'otherShare', 'external', 'remote', 'favorites', 'team'],
          redirectUri: callbackPath(),
          state,
        },
        { target: '_blank', features: 'width=750,height=630' },
      )
      message.info('请在新窗口中完成目录授权')
      return
    }
    const result = await sdk.pickSharedFile({
      title: '选择授权目录',
      okText: '确认授权',
      sidebarGroup: ['myFiles', 'otherShare', 'external', 'remote', 'favorites', 'team'],
      creatable: true,
    })
    if (result?.code !== 0) {
      throw new Error(result?.msg || '目录授权失败')
    }
    message.success('目录授权成功')
    await loadSharedFolders()
  } catch (error) {
    message.error(errorMessage(error, '目录授权失败'))
  } finally {
    authorizing.value = false
  }
}

async function authorizeKnownPath() {
  const path = knownSharedPath.value.trim()
  if (!path) {
    message.warning('请填写要授权的目录路径')
    return
  }
  authorizingKnownPath.value = true
  try {
    await sdk.ready()
    if (sdk.isStandaloneWeb) {
      const state = createAuthState()
      pendingAuthState.value = state
      await sdk.openAppAuth(
        'authorizeSharedFile',
        {
          appName: 'FnTube',
          path,
          redirectUri: callbackPath(),
          state,
        },
        { target: '_blank', features: 'width=750,height=630' },
      )
      message.info('请在新窗口中确认目录授权')
      return
    }
    const result = await sdk.authorizeSharedFile(path)
    if (result?.code !== 0) {
      throw new Error(result?.msg || '目录授权失败')
    }
    message.success('目录授权成功')
    await loadSharedFolders()
  } catch (error) {
    message.error(errorMessage(error, '目录授权失败'))
  } finally {
    authorizingKnownPath.value = false
  }
}

async function loadSharedFolders() {
  loadingSharedFolders.value = true
  try {
    const { data } = await getSharedFolders()
    sharedFolders.value = data.paths || []
    sharedFoldersLoaded.value = true
  } catch (error) {
    message.error(errorMessage(error, '查询授权目录失败'))
  } finally {
    loadingSharedFolders.value = false
  }
}

async function runACLCheck() {
  const paths = aclForm.paths.split(/\r?\n/).map(path => path.trim()).filter(Boolean)
  if (!aclForm.uid || paths.length === 0) {
    message.warning('请填写 UID 和至少一个检查路径')
    return
  }
  checkingACL.value = true
  aclResults.value = []
  try {
    const { data } = await checkACL(aclForm.uid, paths)
    aclResults.value = data
    if (data.length === 0) {
      message.info('接口未返回权限结果')
    }
  } catch (error) {
    message.error(errorMessage(error, '文件权限检查失败'))
  } finally {
    checkingACL.value = false
  }
}

function handleAuthResult(event: MessageEvent) {
  if (event.origin !== window.location.origin || event.data?.type !== 'fntube:shared-auth-result') {
    return
  }
  const result = event.data.result
  if (!pendingAuthState.value || result?.state !== pendingAuthState.value) {
    return
  }
  pendingAuthState.value = ''
  if (result?.status === 'success') {
    message.success('目录授权成功')
    void loadSharedFolders()
  } else {
    message.error(result?.error || '目录授权失败')
  }
}

onMounted(() => window.addEventListener('message', handleAuthResult))
onBeforeUnmount(() => window.removeEventListener('message', handleAuthResult))
</script>

<style scoped>
.system-test-view {
  display: grid;
  gap: 12px;
}

.section-alert {
  margin-bottom: 16px;
}

.known-path-input {
  max-width: 720px;
  margin-top: 16px;
}

.result-block {
  margin-top: 16px;
}

.uid-input {
  width: 180px;
}
</style>
