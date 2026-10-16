<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { FolderOpenOutlined, FontSizeOutlined } from '@ant-design/icons-vue'
import GuangyaFolderPickerDialog, {
  type GuangyaFolderSelection,
} from '@/components/GuangyaFolderPickerDialog.vue'
import {
  createGuangyaCloudTask,
  listGuangyaCloudFolders,
  resolveGuangyaCloudURL,
  type GuangyaCloudFile,
  type GuangyaCloudResolved,
} from '@/api/guangya'
import {
  getStorageMappings,
  type StorageMapping,
} from '@/api/storageconfig'

const LAST_FOLDER_KEY = 'fntube_guangya_cloud_add_folder'
const DEFAULT_FOLDER_NAME = '来自：云添加'
const ROOT_FOLDER: GuangyaFolderSelection = {
  id: '',
  name: '全部文件',
  path: '/',
}

const props = defineProps<{ open: boolean; magnet: string }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const loading = ref(false)
const destinationLoading = ref(false)
const mappingsLoading = ref(false)
const creating = ref(false)
const error = ref('')
const resolved = ref<GuangyaCloudResolved>()
const name = ref('')
const selectedIndexes = ref<number[]>([])
const folderOpen = ref(false)
const selectedFolder = ref<GuangyaFolderSelection>({ ...ROOT_FOLDER })
const storageMappings = ref<StorageMapping[]>([])

interface FileTreeNode {
  key: string
  title: string
  children?: FileTreeNode[]
  isLeaf?: boolean
  disabled?: boolean
  fileIndex?: number
  fileName?: string
  fileSize?: number
  path?: string[]
  isVideo?: boolean
}

function buildFileTree(
  files: GuangyaCloudFile[],
  keyPath = '',
  parentPath: string[] = [],
): FileTreeNode[] {
  return files.map((file, index) => {
    const key = keyPath ? `${keyPath}-${index}` : String(index)
    const filePath = [...parentPath, file.fileName]
    if (file.subfiles?.length) {
      return {
        key,
        title: file.fileName,
        children: buildFileTree(file.subfiles, key, filePath),
      }
    }
    if (file.isDir) {
      return { key, title: file.fileName, isLeaf: true, disabled: true }
    }
    return buildFileNode(file, key, filePath)
  })
}

function buildFileNode(
  file: GuangyaCloudFile,
  key: string,
  filePath: string[],
): FileTreeNode {
  return {
    key,
    title: `${file.fileName} (${bytes(file.fileSize)})`,
    isLeaf: true,
    fileIndex: file.fileIndex ?? 0,
    fileName: file.fileName,
    fileSize: file.fileSize || 0,
    path: filePath,
    isVideo: (file.fileSize || 0) > 100 * 1024 * 1024 && isVideoFile(file),
  }
}

function buildResolvedFileTree(data?: GuangyaCloudResolved): FileTreeNode[] {
  if (!data) return []
  if (data.subfiles?.length) return buildFileTree(data.subfiles)
  if (!data.fileName) return []
  return [
    buildFileNode(
      { fileName: data.fileName, fileSize: data.fileSize, fileIndex: 0 },
      '0',
      [data.fileName],
    ),
  ]
}

function collectFileNodes(nodes: FileTreeNode[]): FileTreeNode[] {
  return nodes.flatMap((node) => [
    ...(node.fileIndex !== undefined ? [node] : []),
    ...(node.children ? collectFileNodes(node.children) : []),
  ])
}

const fileTree = computed(() => buildResolvedFileTree(resolved.value))
const files = computed(() => collectFileNodes(fileTree.value))
const selectedTreeKeys = ref<string[]>([])
const videoExtensions = new Set([
  '3gp',
  'avi',
  'flv',
  'm2ts',
  'm4v',
  'mkv',
  'mov',
  'mp4',
  'mpeg',
  'mpg',
  'rm',
  'rmvb',
  'ts',
  'vob',
  'webm',
  'wmv',
])

function isVideoFile(file: GuangyaCloudFile): boolean {
  const extension = file.fileName.split('.').pop()?.toLowerCase() || ''
  return videoExtensions.has(extension)
}

function bytes(value = 0): string {
  if (!value) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}`
}

function reset() {
  loading.value = false
  destinationLoading.value = false
  mappingsLoading.value = false
  creating.value = false
  error.value = ''
  resolved.value = undefined
  name.value = ''
  selectedIndexes.value = []
  selectedTreeKeys.value = []
  folderOpen.value = false
  selectedFolder.value = { ...ROOT_FOLDER }
  storageMappings.value = []
}

watch(
  () => [props.open, props.magnet] as const,
  ([open]) => {
    if (open) {
      reset()
      void resolve()
      void restoreDestination()
      void loadStorageMappings()
    }
  },
)

async function resolve() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await resolveGuangyaCloudURL(props.magnet)
    resolved.value = data
    name.value = data.fileName || ''
    selectedTreeKeys.value = collectVideoKeys(fileTree.value)
    selectedIndexes.value = selectedTreeKeys.value
      .map((key) => findFileNode(fileTree.value, key)?.fileIndex)
      .filter((index): index is number => index !== undefined)
  } catch (e) {
    error.value = errorMessage(e, '解析磁链失败')
  } finally {
    loading.value = false
  }
}

function collectVideoKeys(nodes: FileTreeNode[]): string[] {
  return nodes.flatMap((node) => {
    const childKeys = node.children ? collectVideoKeys(node.children) : []
    return node.isVideo ? [node.key, ...childKeys] : childKeys
  })
}

function findFileNode(nodes: FileTreeNode[], key: string): FileTreeNode | undefined {
  for (const node of nodes) {
    if (node.key === key) return node
    if (node.children) {
      const result = findFileNode(node.children, key)
      if (result) return result
    }
  }
}

function handleTreeCheck(
  keys: string[] | { checked: string[]; halfChecked: string[] },
) {
  const checkedKeys = Array.isArray(keys) ? keys : keys.checked
  selectedTreeKeys.value = checkedKeys
  const checked = new Set(checkedKeys)
  selectedIndexes.value = files.value
    .filter((node) => checked.has(node.key))
    .map((node) => node.fileIndex)
    .filter((index): index is number => index !== undefined)
}

function errorMessage(error: unknown, fallback: string): string {
  const candidate = error as { message?: string; response?: { data?: { error?: string } } }
  return candidate.response?.data?.error || candidate.message || fallback
}

function openFolderPicker() {
  folderOpen.value = true
}

function selectFolder(selection: GuangyaFolderSelection) {
  selectedFolder.value = selection
}

async function loadStorageMappings() {
  mappingsLoading.value = true
  try {
    const { data } = await getStorageMappings()
    storageMappings.value = data || []
  } catch (e) {
    message.warning(errorMessage(e, '读取存储配置失败'))
  } finally {
    mappingsLoading.value = false
  }
}

function applyStorageMapping(mapping: StorageMapping) {
  const pathParts = mapping.guangya_folder_path.split('/').filter(Boolean)
  selectedFolder.value = {
    id: mapping.guangya_folder_id,
    name: pathParts[pathParts.length - 1] || ROOT_FOLDER.name,
    path: mapping.guangya_folder_path,
  }
}

function isStorageMappingSelected(mapping: StorageMapping): boolean {
  return (
    selectedFolder.value.id === mapping.guangya_folder_id &&
    selectedFolder.value.path === mapping.guangya_folder_path
  )
}

function storedDestination(): GuangyaFolderSelection | undefined {
  const stored = localStorage.getItem(LAST_FOLDER_KEY)
  if (!stored) return
  try {
    const selection = JSON.parse(stored) as Partial<GuangyaFolderSelection>
    if (
      typeof selection.id === 'string' &&
      typeof selection.name === 'string' &&
      typeof selection.path === 'string'
    ) {
      return {
        id: selection.id,
        name: selection.name,
        path: selection.path,
      }
    }
  } catch {
    localStorage.removeItem(LAST_FOLDER_KEY)
  }
}

async function restoreDestination() {
  const stored = storedDestination()
  if (stored) {
    selectedFolder.value = stored
    return
  }

  destinationLoading.value = true
  try {
    let page = 0
    let loaded = 0
    let total = 0
    do {
      const { data } = await listGuangyaCloudFolders('', page, 100)
      const folders = data.list || []
      const defaultFolder = folders.find((folder) => folder.fileName === DEFAULT_FOLDER_NAME)
      if (defaultFolder) {
        selectedFolder.value = {
          id: defaultFolder.fileId,
          name: defaultFolder.fileName,
          path: `/${defaultFolder.fileName}`,
        }
        return
      }
      loaded += folders.length
      total = data.total || loaded
      page += 1
    } while (loaded < total && page < 100)
  } catch (e) {
    message.warning(errorMessage(e, '获取默认保存目录失败，已使用根目录'))
  } finally {
    destinationLoading.value = false
  }
}

function uppercaseName() {
  name.value = name.value.toUpperCase()
}

async function create() {
  if (
    destinationLoading.value ||
    !resolved.value ||
    (files.value.length > 0 && selectedIndexes.value.length === 0)
  ) {
    return
  }
  const selected = new Set(selectedTreeKeys.value)
  const selectedFiles = files.value.filter((node) => selected.has(node.key))
  creating.value = true
  error.value = ''
  try {
    await createGuangyaCloudTask({
      url: resolved.value.url || props.magnet,
      parentId: selectedFolder.value.id,
      newName: name.value.trim(),
      fileIndexes: selectedIndexes.value,
      resolvedIsDirectory: Boolean(resolved.value.subfiles?.length),
      selectedFiles: selectedFiles
        .filter(
          (
            node,
          ): node is FileTreeNode & {
            fileIndex: number
            fileName: string
            path: string[]
          } =>
            node?.fileIndex !== undefined &&
            Boolean(node.fileName) &&
            Boolean(node.path?.length),
        )
        .map((node) => ({
          fileIndex: node.fileIndex,
          fileName: node.fileName,
          fileSize: node.fileSize || 0,
          path: node.path,
        })),
    })
    localStorage.setItem(LAST_FOLDER_KEY, JSON.stringify(selectedFolder.value))
    message.success('已创建光鸭云添加任务')
    emit('update:open', false)
  } catch (e) {
    error.value = errorMessage(e, '创建云添加任务失败')
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <a-modal
    :open="open"
    title="云添加到光鸭"
    :closable="!creating"
    :mask-closable="!creating"
    @cancel="emit('update:open', false)"
  >
    <a-spin :spinning="loading">
      <a-alert v-if="error" type="error" :message="error" show-icon />
      <template v-if="resolved">
        <a-form layout="vertical">
          <a-form-item label="名称">
            <a-input-group compact class="name-input">
              <a-input v-model:value="name" :disabled="creating" />
              <a-tooltip title="转大写">
                <a-button
                  aria-label="转大写"
                  :disabled="creating || !name"
                  @click="uppercaseName"
                >
                  <template #icon><FontSizeOutlined /></template>
                </a-button>
              </a-tooltip>
            </a-input-group>
          </a-form-item>
          <a-form-item v-if="files.length" label="选择文件">
            <a-tree
              checkable
              block-node
              default-expand-all
              :tree-data="fileTree"
              :checked-keys="selectedTreeKeys"
              :disabled="creating"
              @check="handleTreeCheck"
            />
          </a-form-item>
        </a-form>
      </template>
    </a-spin>
    <template #footer>
      <div class="modal-footer">
        <div v-if="mappingsLoading || storageMappings.length" class="storage-mappings">
          <span class="storage-mappings-label">路径映射</span>
          <a-spin v-if="mappingsLoading" size="small" />
          <div v-else class="storage-mapping-list">
            <a-tooltip
              v-for="mapping in storageMappings"
              :key="mapping.id"
              :title="mapping.guangya_folder_path"
            >
              <a-button
                size="small"
                :type="isStorageMappingSelected(mapping) ? 'primary' : 'default'"
                :disabled="creating"
                @click="applyStorageMapping(mapping)"
              >
                {{ mapping.name }}
              </a-button>
            </a-tooltip>
          </div>
        </div>
        <div class="modal-footer-actions">
          <div class="destination">
            <span>保存到</span>
            <a-button
              :loading="destinationLoading"
              :disabled="creating"
              @click="openFolderPicker"
            >
              <template #icon><FolderOpenOutlined /></template>
              {{ selectedFolder.name }}
            </a-button>
          </div>
          <a-space>
            <a-button :disabled="creating" @click="emit('update:open', false)">取消</a-button>
            <a-button
              type="primary"
              :loading="creating"
              :disabled="
                loading ||
                destinationLoading ||
                !resolved ||
                (files.length > 0 && selectedIndexes.length === 0)
              "
              @click="create"
            >
              添加到云盘
            </a-button>
          </a-space>
        </div>
      </div>
    </template>
  </a-modal>

  <GuangyaFolderPickerDialog
    v-model:open="folderOpen"
    title="选择目标文件夹"
    ok-text="选择此文件夹"
    :selected-id="selectedFolder.id"
    :selected-path="selectedFolder.path"
    @select="selectFolder"
  />
</template>

<style scoped lang="scss">
.modal-footer {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.modal-footer-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.storage-mappings {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  text-align: left;
}

.storage-mappings-label {
  flex: none;
  line-height: 24px;
}

.storage-mapping-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.destination {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.name-input {
  display: flex;
}

.name-input .ant-input {
  flex: 1;
  min-width: 0;
}

.file-size {
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}

</style>
