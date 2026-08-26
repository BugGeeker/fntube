<template>
  <a-card title="飞牛影视配置">
    <a-form :model="form" layout="vertical" @finish="handleSave">
      <a-form-item label="服务端地址" name="host" required>
        <a-input v-model:value="form.host" placeholder="如 http://127.0.0.1:5666/v" allow-clear />
      </a-form-item>
      <a-form-item label="用户名" name="username" required>
        <a-input v-model:value="form.username" placeholder="飞牛影视用户名" allow-clear />
      </a-form-item>
      <a-form-item label="密码" name="password" required>
        <a-input-password v-model:value="form.password" placeholder="飞牛影视密码" allow-clear />
      </a-form-item>
      <a-form-item label="访问码" name="access_code">
        <a-input v-model:value="form.access_code" placeholder="未开启访问码则留空" allow-clear />
      </a-form-item>
      <a-form-item label="外网播放地址" name="play_host">
        <a-input v-model:value="form.play_host" placeholder="如 http://your-domain:5666/v" allow-clear />
      </a-form-item>
      <a-form-item label="同步媒体库" name="sync_libraries">
        <a-space-compact block>
          <a-select v-model:value="selectedLibraries" mode="multiple" placeholder="点击右侧按钮加载媒体库列表" allow-clear
            :options="libraryOptions" :field-names="{ label: 'name', value: 'id' }" option-filter-prop="name"
            style="width: 100%" />
          <a-button :loading="loadingLibs" @click="handleLoadLibraries">
            加载媒体库列表
          </a-button>
        </a-space-compact>

        <span style="margin-left: 8px; color: rgba(0,0,0,0.45)">
          选中“全部媒体库”即同步所有媒体库，或按需勾选
        </span>
      </a-form-item>
      <a-space>
        <a-button type="primary" html-type="submit" :loading="saving">保存配置</a-button>
        <a-button :loading="testing" @click="handleTest">测试连接</a-button>
      </a-space>
    </a-form>

    <a-alert v-if:="testResult" style="margin-top: 16px" :type="testResult.success ? 'success' : 'error'"
      :message="testResult.message" show-icon />
  </a-card>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed, watch } from 'vue'
import { message } from 'ant-design-vue'
import { useTrimMediaStore } from '@/stores/trimmedia'
import { testConnection, type TrimMediaConfig, type Library } from '@/api/trimmedia'

const store = useTrimMediaStore()

const form = reactive<TrimMediaConfig>({
  host: '',
  username: '',
  password: '',
  access_code: '',
  play_host: '',
  sync_libraries: '',
})

const saving = ref(false)
const testing = ref(false)
const testResult = ref<{ success: boolean; message: string } | null>(null)

const libraries = ref<Library[]>([])
const loadingLibs = ref(false)
const selectedLibraries = ref<string[]>([])

const ALL_LIBRARIES_ID = 'all'
const libraryOptions = computed(() => {
  const opts: { id: string; name: string }[] = [
    { id: ALL_LIBRARIES_ID, name: '全部媒体库' },
  ]
  return opts.concat(libraries.value)
})

function syncSelectedToForm() {
  if (selectedLibraries.value.includes(ALL_LIBRARIES_ID)) {
    form.sync_libraries = JSON.stringify([ALL_LIBRARIES_ID])
  } else {
    form.sync_libraries = JSON.stringify(selectedLibraries.value)
  }
}

function parseFormToSelected() {
  try {
    const parsed = JSON.parse(form.sync_libraries || '[]')
    if (Array.isArray(parsed)) {
      selectedLibraries.value = parsed as string[]
    } else {
      selectedLibraries.value = []
    }
  } catch {
    selectedLibraries.value = []
  }
}

watch(selectedLibraries, (val) => {
  // 选中“全部媒体库”时，清除其他选项
  if (val.includes(ALL_LIBRARIES_ID) && val.length > 1) {
    selectedLibraries.value = [ALL_LIBRARIES_ID]
  }
  syncSelectedToForm()
})

onMounted(async () => {
  try {
    await store.fetchConfig()
    if (store.config) {
      Object.assign(form, store.config)
      if(form.host && form.username && form.password) {
        handleLoadLibraries()
      }
    }
  } catch {
    // 配置尚未保存
  }
  parseFormToSelected()
})

async function handleSave() {
  if (!form.host || !form.username || !form.password) {
    message.warning('请填写服务端地址、用户名和密码')
    return
  }
  saving.value = true
  try {
    await store.saveConfigData({ ...form })
    message.success('配置已保存，服务将在后台重连')
  } catch (e: any) {
    message.error('保存失败: ' + (e?.message || ''))
  } finally {
    saving.value = false
  }
}

async function handleTest() {
  if (!form.host || !form.username || !form.password) {
    message.warning('请填写服务端地址、用户名和密码')
    return
  }
  testing.value = true
  testResult.value = null
  try {
    const result = await store.testConnectionData({ ...form })
    testResult.value = {
      success: true,
      message: `连接成功！前端版本: ${result.version?.frontend || '未知'}, 后端版本: ${result.version?.backend || '未知'}`,
    }
  } catch (e: any) {
    testResult.value = { success: false, message: '连接失败: ' + (e?.message || '请检查配置') }
  } finally {
    testing.value = false
  }
}

async function handleLoadLibraries() {
  if (!form.host || !form.username || !form.password) {
    message.warning('请填写服务端地址、用户名和密码')
    return
  }
  loadingLibs.value = true
  try {
    const { data } = await testConnection({ ...form })
    libraries.value = data.libraries || []
    // 保留已选中但加载结果中仍存在的媒体库 id；'all' 不自动选中
    const validIds = new Set(libraries.value.map((lib) => lib.id))
    const prev = selectedLibraries.value.filter(
      (id) => id === ALL_LIBRARIES_ID || validIds.has(id),
    )
    selectedLibraries.value = prev
    // message.success(`已加载 ${libraries.value.length} 个媒体库`)
  } catch (e: any) {
    message.error('加载媒体库失败: ' + (e?.message || '请检查配置'))
  } finally {
    loadingLibs.value = false
  }
}
</script>
