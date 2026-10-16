<template>
  <a-card title="影片搜索">
    <template #extra>
      <a-space>
        <a-select
          v-model:value="selectedSource"
          class="source-select"
          :options="sourceOptions"
          :loading="loadingSources"
          aria-label="搜索源"
          @change="handleSourceChange"
        />
        <a-tooltip title="搜索设置">
          <a-button shape="circle" aria-label="搜索设置" @click="openSettings">
            <template #icon><SettingOutlined /></template>
          </a-button>
        </a-tooltip>
      </a-space>
    </template>

    <div class="search-bar">
      <a-input-search
        v-model:value="keyword"
        placeholder="输入影片番号"
        enter-button="搜索"
        size="large"
        allow-clear
        :loading="searching"
        @search="handleSearch"
      />
    </div>

    <div v-if="searched && results.length > 0" class="movie-info-card">
      <div class="movie-cover">
        <img
          v-if="movieInfo.cover_url && !uiStore.hideImages"
          :src="movieInfo.cover_url"
          :alt="movieInfo.title || movieInfo.name"
        />
        <span v-else>{{ movieInfo.title || movieInfo.name }}</span>
      </div>
      <div class="movie-info-content">
        <h3>{{ movieInfo.title || movieInfo.name || '-' }}</h3>
        <div class="movie-info-grid">
          <div class="movie-info-item"><span>番号</span><strong>{{ movieInfo.number || '-' }}</strong></div>
          <div class="movie-info-item"><span>片商</span><strong>{{ movieInfo.maker || '-' }}</strong></div>
          <div class="movie-info-item"><span>发行日期</span><strong>{{ movieInfo.release_date || '-' }}</strong></div>
          <div class="movie-info-item"><span>时长</span><strong>{{ movieInfo.runtime ? `${movieInfo.runtime} 分钟` : '-' }}</strong></div>
          <div class="movie-info-item"><span>系列</span><strong>{{ movieInfo.series || '-' }}</strong></div>
          <div class="movie-info-item"><span>发行商</span><strong>{{ movieInfo.label || '-' }}</strong></div>
          <div class="movie-info-item"><span>导演</span><strong>{{ movieInfo.director || '-' }}</strong></div>
          <div class="movie-info-item movie-info-wide">
            <span>类别</span>
            <strong>{{ movieInfo.categories?.length ? movieInfo.categories.join('、') : '-' }}</strong>
          </div>
          <div class="movie-info-item movie-info-wide">
            <span>演员</span>
            <strong>{{ movieInfo.actors?.length ? movieInfo.actors.join('、') : '-' }}</strong>
          </div>
        </div>
      </div>
    </div>

    <a-spin :spinning="searching">
      <a-empty v-if="searched && results.length === 0" description="未找到磁链" />
      <div v-else-if="!searched" class="initial-state">
        <SearchOutlined />
        <span>输入影片番号开始搜索</span>
      </div>

      <div v-else class="desktop-results">
        <a-table
          :columns="columns"
          :data-source="results"
          :pagination="false"
          :row-key="resultRowKey"
          table-layout="fixed"
          :scroll="{ x: 860 }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="result-name">{{ record.name }}</div>
              <div v-if="record.tags?.length" class="result-tags">
                <a-tag v-for="tag in record.tags" :key="tag">{{ tag }}</a-tag>
              </div>
            </template>
            <template v-else-if="column.key === 'files'">
              {{ record.files || '-' }}
            </template>
            <template v-else-if="column.key === 'action'">
              <a-space>
                <a-tooltip title="复制磁链">
                  <a-button shape="circle" @click="copyMagnet(record.magnet)">
                    <template #icon><CopyOutlined /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip title="云添加到光鸭">
                  <a-button type="primary" shape="circle" @click="openCloudAdd(record.magnet)">
                    <template #icon><ExportOutlined /></template>
                  </a-button>
                </a-tooltip>
              </a-space>
            </template>
          </template>
        </a-table>
      </div>

      <div v-if="searched && results.length > 0" class="mobile-results">
        <div v-for="record in results" :key="resultRowKey(record)" class="mobile-result">
          <div class="mobile-result-header">
            <span class="result-name">{{ record.name }}</span>
            <a-space>
              <a-button size="small" shape="circle" aria-label="复制磁链" @click="copyMagnet(record.magnet)">
                <template #icon><CopyOutlined /></template>
              </a-button>
              <a-button type="primary" size="small" shape="circle" aria-label="云添加到光鸭"
                @click="openCloudAdd(record.magnet)">
                <template #icon><ExportOutlined /></template>
              </a-button>
            </a-space>
          </div>
          <div class="mobile-result-meta">
            <span>{{ record.size || '未知大小' }}</span>
            <span>{{ record.date || '未知日期' }}</span>
            <span v-if="record.files">{{ record.files }} 个文件</span>
          </div>
          <div v-if="record.tags?.length" class="result-tags">
            <a-tag v-for="tag in record.tags" :key="tag">{{ tag }}</a-tag>
          </div>
        </div>
      </div>
    </a-spin>

    <a-modal
      v-model:open="settingsVisible"
      title="搜索设置"
      :confirm-loading="savingSettings"
      ok-text="保存"
      cancel-text="取消"
      @ok="saveSettings"
    >
      <a-form layout="vertical" :model="settingsForm">
        <a-form-item label="JavDB 域名" required>
          <a-input v-model:value="settingsForm.javdb_domain" addon-before="https://" allow-clear />
        </a-form-item>
        <a-form-item label="JavBus 域名" required>
          <a-input v-model:value="settingsForm.javbus_domain" addon-before="https://" allow-clear />
        </a-form-item>
      </a-form>
    </a-modal>
    <GuangyaCloudAddDialog v-model:open="cloudAddVisible" :magnet="cloudAddMagnet" />
  </a-card>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import { CopyOutlined, ExportOutlined, SearchOutlined, SettingOutlined } from '@ant-design/icons-vue'
import {
  getMovieSearchConfig,
  getMovieSearchSources,
  saveMovieSearchConfig,
  searchMovies,
  type MovieSearchConfig,
  type MovieSearchResult,
  type MovieSearchSource,
} from '@/api/moviesearch'
import GuangyaCloudAddDialog from '@/components/GuangyaCloudAddDialog.vue'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()
const keyword = ref('')
const selectedSource = ref('javbus')
const sources = ref<MovieSearchSource[]>([])
const results = ref<MovieSearchResult[]>([])
const loadingSources = ref(false)
const searching = ref(false)
const searched = ref(false)
const settingsVisible = ref(false)
const savingSettings = ref(false)
const cloudAddVisible = ref(false)
const cloudAddMagnet = ref('')
const config = reactive<MovieSearchConfig>({
  source: 'javbus',
  javdb_domain: 'www.javdb.com',
  javbus_domain: 'www.javbus.com',
})
const settingsForm = reactive({
  javdb_domain: 'www.javdb.com',
  javbus_domain: 'www.javbus.com',
})

const sourceOptions = computed(() =>
  sources.value.map((source) => ({ value: source.name, label: source.label })),
)

const movieInfo = computed(() => results.value[0] || ({} as MovieSearchResult))

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '大小', dataIndex: 'size', key: 'size', width: 110 },
  { title: '文件数', dataIndex: 'files', key: 'files', width: 90 },
  { title: '日期', dataIndex: 'date', key: 'date', width: 120 },
  { title: '操作', key: 'action', width: 110, fixed: 'right' },
]

function errorMessage(error: unknown, fallback: string): string {
  const candidate = error as {
    response?: { data?: { error?: string } }
    message?: string
  }
  return candidate.response?.data?.error || candidate.message || fallback
}

async function loadSettings() {
  loadingSources.value = true
  try {
    const [sourcesResponse, configResponse] = await Promise.all([
      getMovieSearchSources(),
      getMovieSearchConfig(),
    ])
    sources.value = sourcesResponse.data.data || []
    Object.assign(config, configResponse.data)
    if (sources.value.some((source) => source.name === configResponse.data.source)) {
      selectedSource.value = configResponse.data.source
    } else if (sources.value.length > 0) {
      selectedSource.value = sources.value[0].name
    }
  } catch (error) {
    message.error(errorMessage(error, '加载搜索源失败'))
  } finally {
    loadingSources.value = false
  }
}

async function handleSourceChange(source: string) {
  results.value = []
  searched.value = false
  try {
    const { data } = await saveMovieSearchConfig({ ...config, source })
    Object.assign(config, data.config)
    message.success('默认搜索源已更新')
  } catch (error) {
    selectedSource.value = config.source
    message.error(errorMessage(error, '保存搜索源失败'))
  }
}

function openSettings() {
  settingsForm.javdb_domain = config.javdb_domain
  settingsForm.javbus_domain = config.javbus_domain
  settingsVisible.value = true
}

async function saveSettings() {
  if (!settingsForm.javdb_domain.trim() || !settingsForm.javbus_domain.trim()) {
    message.warning('请填写 JavDB 和 JavBus 域名')
    return
  }

  savingSettings.value = true
  try {
    const { data } = await saveMovieSearchConfig({
      source: selectedSource.value,
      javdb_domain: settingsForm.javdb_domain.trim(),
      javbus_domain: settingsForm.javbus_domain.trim(),
    })
    Object.assign(config, data.config)
    settingsVisible.value = false
    results.value = []
    searched.value = false
    message.success('搜索设置已保存')
  } catch (error) {
    message.error(errorMessage(error, '保存搜索设置失败'))
  } finally {
    savingSettings.value = false
  }
}

async function handleSearch() {
  const query = keyword.value.trim()
  if (!query) {
    message.warning('请输入影片番号')
    return
  }
  if (!selectedSource.value) {
    message.warning('请选择搜索源')
    return
  }

  searching.value = true
  searched.value = true
  try {
    const { data } = await searchMovies(selectedSource.value, query)
    results.value = data.data || []
  } catch (error) {
    results.value = []
    message.error(errorMessage(error, '影片搜索失败'))
  } finally {
    searching.value = false
  }
}

async function copyMagnet(magnet: string) {
  try {
    await navigator.clipboard.writeText(magnet)
    message.success('磁链已复制')
  } catch {
    message.error('复制失败，请检查浏览器剪贴板权限')
  }
}

function openCloudAdd(magnet: string) {
  cloudAddMagnet.value = magnet
  cloudAddVisible.value = true
}

function resultRowKey(record: MovieSearchResult): string {
  return record.hash || record.magnet
}

onMounted(loadSettings)
</script>

<style scoped lang="scss">
.source-select {
  width: 140px;
}

.search-bar {
  max-width: 720px;
  margin: 0 auto 24px;
}

.movie-info-card {
  display: flex;
  gap: 20px;
  margin: 0 auto 24px;
  max-width: 980px;
  padding: 16px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  background: #fff;
}

.movie-cover {
  display: flex;
  width: 150px;
  min-width: 150px;
  height: 210px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  color: rgba(0, 0, 0, 0.45);
  text-align: center;
  background: #f5f5f5;
}

.movie-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.movie-info-content {
  min-width: 0;
  flex: 1;
}

.movie-info-content h3 {
  margin: 0 0 14px;
  overflow-wrap: anywhere;
  font-size: 20px;
}

.movie-info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 20px;
}

.movie-info-item {
  display: grid;
  grid-template-columns: 70px minmax(0, 1fr);
  gap: 8px;
  min-width: 0;
  line-height: 1.6;
}

.movie-info-item span {
  color: rgba(0, 0, 0, 0.45);
}

.movie-info-item strong {
  min-width: 0;
  overflow-wrap: anywhere;
  font-weight: 400;
}

.movie-info-wide {
  grid-column: 1 / -1;
}

.initial-state {
  display: flex;
  min-height: 240px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.initial-state :deep(svg) {
  font-size: 40px;
}

.result-name {
  overflow-wrap: anywhere;
  font-weight: 600;
}

.result-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 8px;
}

.result-tags :deep(.ant-tag) {
  margin-inline-end: 0;
}

.mobile-results {
  display: none;
}

@media (max-width: 639px) {
  .source-select {
    width: 112px;
  }

  .search-bar {
    margin-bottom: 16px;
  }

  .movie-info-card {
    flex-direction: column;
    gap: 14px;
    margin-bottom: 16px;
    padding: 12px;
  }

  .movie-cover {
    width: 100%;
    min-width: 0;
    height: 220px;
  }

  .movie-info-content h3 {
    font-size: 18px;
  }

  .movie-info-grid {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .movie-info-wide {
    grid-column: auto;
  }

  .desktop-results {
    display: none;
  }

  .mobile-results {
    display: block;
  }

  .mobile-result {
    padding: 14px 0;
    border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  }

  .mobile-result:first-child {
    padding-top: 0;
  }

  .mobile-result:last-child {
    border-bottom: 0;
  }

  .mobile-result-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
  }

  .mobile-result-header .result-name {
    min-width: 0;
  }

  .mobile-result-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 12px;
    margin-top: 8px;
    color: rgba(0, 0, 0, 0.45);
    font-size: 12px;
  }
}
</style>
