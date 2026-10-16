<script setup lang="ts">
import { ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { FolderAddOutlined } from '@ant-design/icons-vue'
import {
  createGuangyaCloudFolder,
  listGuangyaCloudFolders,
  type GuangyaCloudFolder,
} from '@/api/guangya'

export interface GuangyaFolderSelection {
  id: string
  path: string
  name: string
}

interface FolderTreeNode {
  key: string
  folderId: string
  title: string
  path: string
  isLeaf?: boolean
  children?: FolderTreeNode[]
  loaded?: boolean
}

const ROOT_KEY = '__guangya_root__'
const ROOT_NAME = '全部文件'

const props = withDefaults(
  defineProps<{
    open: boolean
    selectedId?: string
    selectedPath?: string
    title?: string
    okText?: string
  }>(),
  {
    selectedId: '',
    selectedPath: '/',
    title: '选择光鸭云盘目录',
    okText: '选择此目录',
  },
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  select: [selection: GuangyaFolderSelection]
}>()

const loading = ref(false)
const error = ref('')
const treeData = ref<FolderTreeNode[]>([])
const treeVersion = ref(0)
const selectedKeys = ref<string[]>([ROOT_KEY])
const expandedKeys = ref<string[]>([ROOT_KEY])
const pendingSelection = ref<GuangyaFolderSelection>(rootSelection())
const createFolderOpen = ref(false)
const creatingFolder = ref(false)
const newFolderName = ref('')

function rootSelection(): GuangyaFolderSelection {
  return {
    id: '',
    path: '/',
    name: ROOT_NAME,
  }
}

function rootNode(): FolderTreeNode {
  return {
    key: ROOT_KEY,
    folderId: '',
    title: ROOT_NAME,
    path: '/',
    isLeaf: false,
  }
}

function errorMessage(cause: unknown, fallback: string): string {
  const candidate = cause as { message?: string; response?: { data?: { error?: string } } }
  return candidate.response?.data?.error || candidate.message || fallback
}

function folderNode(folder: GuangyaCloudFolder, parentPath: string): FolderTreeNode {
  return {
    key: folder.fileId,
    folderId: folder.fileId,
    title: folder.fileName,
    path: joinFolderPath(parentPath, folder.fileName),
    isLeaf: false,
  }
}

function joinFolderPath(parentPath: string, name: string): string {
  return parentPath === '/' ? `/${name}` : `${parentPath}/${name}`
}

async function fetchFolders(parentId: string): Promise<GuangyaCloudFolder[]> {
  const folders: GuangyaCloudFolder[] = []
  let page = 0
  let total = 0
  do {
    const { data } = await listGuangyaCloudFolders(parentId, page, 100)
    folders.push(...(data.list || []))
    total = data.total || folders.length
    page += 1
  } while (folders.length < total && page < 100)
  return folders
}

async function loadChildren(node: FolderTreeNode) {
  if (node.loaded) return
  loading.value = true
  error.value = ''
  try {
    const folders = await fetchFolders(node.folderId)
    node.children = folders.map((folder) => folderNode(folder, node.path))
    node.loaded = true
    if (!folders.length) node.isLeaf = true
  } catch (cause) {
    error.value = errorMessage(cause, '加载光鸭云盘目录失败')
    throw cause
  } finally {
    loading.value = false
  }
}

async function loadFolderTree(node: { dataRef?: FolderTreeNode }) {
  if (!node.dataRef) return
  await loadChildren(node.dataRef)
  treeData.value = [...treeData.value]
}

function findNode(nodes: FolderTreeNode[], key: string): FolderTreeNode | undefined {
  for (const node of nodes) {
    if (node.key === key) return node
    if (node.children) {
      const result = findNode(node.children, key)
      if (result) return result
    }
  }
}

function selectFolderNode(keys: Array<string | number>) {
  const key = String(keys[0] ?? '')
  if (!key) return
  const node = findNode(treeData.value, key)
  if (!node) return
  selectedKeys.value = [key]
  pendingSelection.value = {
    id: node.folderId,
    path: node.path,
    name: node.title,
  }
}

function close() {
  closeCreateFolder()
  emit('update:open', false)
}

function confirm() {
  emit('select', pendingSelection.value)
  close()
}

function openCreateFolder() {
  newFolderName.value = ''
  createFolderOpen.value = true
}

function closeCreateFolder() {
  createFolderOpen.value = false
  newFolderName.value = ''
}

async function createFolder() {
  const name = newFolderName.value.trim()
  if (!name) {
    message.warning('请输入文件夹名称')
    return
  }

  creatingFolder.value = true
  try {
    await createGuangyaCloudFolder(name, pendingSelection.value.id)
    closeCreateFolder()
    message.success('文件夹已创建')

    const selectedKey = pendingSelection.value.id || ROOT_KEY
    const parent = findNode(treeData.value, selectedKey)
    if (parent) {
      parent.loaded = false
      parent.isLeaf = false
      parent.children = undefined
      await loadChildren(parent)
      treeData.value = [...treeData.value]
      if (!expandedKeys.value.includes(selectedKey)) {
        expandedKeys.value = [...expandedKeys.value, selectedKey]
      }
    }
  } catch (cause) {
    if (createFolderOpen.value) {
      message.error(errorMessage(cause, '创建文件夹失败'))
    }
  } finally {
    creatingFolder.value = false
  }
}

async function findSelectedFolder(
  root: FolderTreeNode,
  selectedId: string,
  selectedPath: string,
): Promise<{ node: FolderTreeNode; expanded: string[] } | undefined> {
  const segments = selectedPath
    .split(/[\\/]/)
    .map((segment) => segment.trim())
    .filter(Boolean)
  if (!segments.length) {
    return selectedId ? undefined : { node: root, expanded: [ROOT_KEY] }
  }

  const expanded = [ROOT_KEY]
  let current = root
  for (let index = 0; index < segments.length; index += 1) {
    await loadChildren(current)
    const isTarget = index === segments.length - 1
    const children = current.children || []
    const next = isTarget && selectedId
      ? children.find((child) => child.folderId === selectedId)
      : children.find((child) => child.title === segments[index])
    if (!next) return

    current = next
    if (!isTarget) expanded.push(current.key)
  }

  if (selectedId && current.folderId !== selectedId) return
  return { node: current, expanded }
}

async function initialize() {
  const version = ++treeVersion.value
  const root = rootNode()
  treeData.value = [root]
  expandedKeys.value = []
  error.value = ''
  closeCreateFolder()

  const selectedId = props.selectedId || ''
  const selectedPath = props.selectedPath || '/'
  selectedKeys.value = [selectedId || ROOT_KEY]
  pendingSelection.value = {
    id: selectedId,
    path: selectedPath,
    name: selectedPath.split('/').filter(Boolean).pop() || ROOT_NAME,
  }

  try {
    await loadChildren(root)
    const selected = await findSelectedFolder(root, selectedId, selectedPath)
    if (version !== treeVersion.value) return
    treeData.value = [{ ...root }]
    if (selected) {
      selectedKeys.value = [selected.node.key]
      pendingSelection.value = {
        id: selected.node.folderId,
        path: selected.node.path,
        name: selected.node.title,
      }
      expandedKeys.value = selected.expanded
    } else {
      expandedKeys.value = [ROOT_KEY]
    }
  } catch {
    // The error is displayed in the dialog and can be retried by reopening it.
    if (version === treeVersion.value) expandedKeys.value = [ROOT_KEY]
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) void initialize()
  },
  { immediate: true },
)
</script>

<template>
  <a-modal
    :open="open"
    :title="title"
    @cancel="close"
  >
    <a-alert v-if="error" type="error" :message="error" show-icon />
    <a-spin :spinning="loading">
      <a-tree
        :key="treeVersion"
        v-model:expanded-keys="expandedKeys"
        block-node
        :tree-data="treeData"
        :selected-keys="selectedKeys"
        :load-data="loadFolderTree"
        @select="selectFolderNode"
      />
    </a-spin>
    <template #footer>
      <div class="folder-picker-footer">
        <a-button :disabled="loading" @click="openCreateFolder">
          <template #icon><FolderAddOutlined /></template>
          新建文件夹
        </a-button>
        <a-space>
          <a-button @click="close">取消</a-button>
          <a-button type="primary" :disabled="loading || !!error" @click="confirm">
            {{ okText }}
          </a-button>
        </a-space>
      </div>
    </template>
  </a-modal>

  <a-modal
    v-model:open="createFolderOpen"
    title="新建文件夹"
    ok-text="创建"
    :confirm-loading="creatingFolder"
    @ok="createFolder"
    @cancel="closeCreateFolder"
  >
    <a-form layout="vertical">
      <a-form-item label="文件夹名称" required>
        <a-input
          v-model:value="newFolderName"
          placeholder="请输入文件夹名称"
          :disabled="creatingFolder"
          @press-enter="createFolder"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<style scoped lang="scss">
.folder-picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
</style>
