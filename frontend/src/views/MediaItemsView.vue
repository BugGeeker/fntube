<template>
  <div>
    <!-- 媒体条目列表 -->
    <a-card :title="libraryName || '加载中...'">
      <template #extra>
        <a-button @click="backToLibraries">返回媒体库</a-button>
      </template>
      <a-spin :spinning="store.loading">
        <a-row :gutter="[16, 16]">
          <a-col v-for="item in store.items" :key="item.guid" :xs="24" :sm="12" :md="8" :lg="6" :xl="4">
            <a-card hoverable size="small" @click="showDetail(item)" style="overflow: hidden">
              <template #cover>
                <img v-if="!uiStore.hideImages && item.poster" :src="proxyImage(item.poster)"
                  style="height: 200px; width: 100%; object-fit: cover" alt="poster" />
                <div v-else
                  style="height: 200px; display: flex; align-items: center; justify-content: center; background: #f5f5f5">
                  <span style="color: #999; font-size: 24px">{{ item.title?.charAt(0) || '?' }}</span>
                </div>
              </template>
              <a-card-meta>
                <template #title>
                  <div style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis">{{ item.title }}</div>
                </template>
                <template #description>
                  <span>{{ itemYear(item) || '未知年份' }}</span>
                  <a-tag v-if="item.type === 'TV'" color="blue" style="margin-left: 8px">剧集</a-tag>
                  <a-tag v-else-if="item.type === 'Movie'" color="orange" style="margin-left: 8px">电影</a-tag>
                </template>
              </a-card-meta>
            </a-card>
          </a-col>
        </a-row>
        <a-empty v-if="!store.loading && store.items.length === 0" description="该媒体库暂无内容" />
      </a-spin>
    </a-card>

    <!-- 媒体详情弹窗 -->
    <a-modal v-model:open="detailVisible" width="1000px" :title="store.currentItem?.title || '详情'" :footer="null"
      :body-style="{ maxHeight: '80vh', overflow: 'auto' }">
      <template v-if="store.currentItem">
        <!-- 顶部图片横向布局 -->
        <div
          v-if="!uiStore.hideImages && (store.currentItem.backdrop || store.currentItem.poster || store.currentItem.logo)"
          style="display: flex; gap: 12px; margin-bottom: 16px;">
          <!-- 封面 -->
          <div style="flex: 1; min-width: 0;">
            <img :src="proxyImage(store.currentItem.poster)"
              style="width: 100%; height: 200px; object-fit: contain; border-radius: 8px;" alt="poster" />
          </div>
          <!-- 背景图 -->
          <div style="flex: 1; min-width: 0;">
            <img :src="proxyImage(store.currentItem.backdrop)"
              style="width: 100%; height: 200px; object-fit: contain; border-radius: 8px;" alt="backdrop" />
          </div>

          <!-- Logo -->
          <div style="flex: 1; min-width: 0;">
            <img :src="proxyImage(store.currentItem.logo)"
              style="width: 100%; height: 200px; object-fit: contain; border-radius: 8px; background: #1a1a2e;"
              alt="logo" />
          </div>
        </div>

        <div style="margin-bottom: 16px">
          <p v-if="store.currentItem.original_title && store.currentItem.original_title !== store.currentItem.title"
            style="color: #999">
            {{ store.currentItem.original_title }}
          </p>
          <a-space>
            <a-tag>{{ itemYear(store.currentItem) || '未知年份' }}</a-tag>
            <a-tag v-if="store.currentItem.type === 'TV'" color="blue">剧集</a-tag>
            <a-tag v-else-if="store.currentItem.type === 'Movie'" color="orange">电影</a-tag>
          </a-space>
        </div>

        <!-- 简介 -->
        <div v-if="store.currentItem.overview" style="margin-bottom: 16px">
          <p style="color: #555; line-height: 1.6">{{ store.currentItem.overview }}</p>
        </div>

        <!-- 演员列表 -->
        <div v-if="store.persons.length > 0" style="margin-bottom: 16px">
          <h4>演员</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="p in store.persons" :key="p.person_guid">
              <a-list-item-meta>
                <template #avatar>
                  <a-avatar v-if="!uiStore.hideImages && p.profile_path" :src="proxyImage(p.profile_path)" :size="40" />
                  <a-avatar v-else :size="40">{{ p.name?.charAt(0) || '?' }}</a-avatar>
                </template>
                <template #title>
                  <span>{{ p.name }}</span>
                  <a-tag v-if="p.job" color="blue" style="margin-left: 8px">{{ p.job }}</a-tag>
                </template>
                <template #description>
                  <span v-if="p.role">饰 {{ p.role }}</span>
                  <span v-else-if="p.job">{{ p.job }}</span>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </a-list>
        </div>

        <!-- 剧集季列表 -->
        <div v-if="store.currentItem.type === 'TV' && store.seasons.length > 0">
          <h4>季列表</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="season in store.seasons" :key="season.guid" @click="loadEpisodes(season.guid)"
              style="cursor: pointer">
              <a-list-item-meta :title="season.title || `第 ${season.season_number} 季`" />
            </a-list-item>
          </a-list>
        </div>

        <!-- 集列表 -->
        <div v-if="store.episodes.length > 0" style="margin-top: 16px">
          <h4>剧集列表</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="ep in store.episodes" :key="ep.guid">
              <a-list-item-meta :title="`S${ep.season_number}E${ep.episode_number} - ${ep.title || ''}`" />
            </a-list-item>
          </a-list>
        </div>

        <div style="margin-top: 24px">
          <a-space>
            <a-button type="primary" :loading="gettingURL" @click="handlePlay">
              <template #icon>
                <PlayCircleOutlined />
              </template>
              播放
            </a-button>
            <a-button :loading="editLoading" @click="handleEdit">
              <template #icon>
                <EditOutlined />
              </template>
              编辑
            </a-button>
          </a-space>
        </div>
      </template>
    </a-modal>

    <!-- 编辑弹窗 -->
    <a-modal v-model:open="editVisible" width="900px" title="编辑媒体信息"
      :body-style="{ maxHeight: '80vh', overflow: 'auto' }">
      <a-spin :spinning="editLoading">
        <a-form v-if="editForm" layout="vertical">
          <a-form-item label="海报">
            <a-flex gap="middle" align="center">
              <a-input v-model:value="editForm.posters" />
              <LockButton v-model:locked="editForm.posters_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="背景图">
            <a-flex gap="middle" align="center">
              <a-input v-model:value="editForm.backdrops" />
              <LockButton v-model:locked="editForm.backdrops_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="Logo">
            <a-flex gap="middle" align="center">
              <a-input v-model:value="editForm.logos" />
              <LockButton v-model:locked="editForm.logos_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="标题">
            <a-flex gap="middle" align="center">
              <a-input v-model:value="editForm.title" />
              <LockButton v-model:locked="editForm.title_locked" />
            </a-flex>
          </a-form-item>

          <a-form-item label="简介">
            <a-flex gap="middle" align="center">
              <a-textarea v-model:value="editForm.overview" :rows="4" />
              <LockButton v-model:locked="editForm.overview_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="评分">
            <a-flex gap="middle" align="center">
              <a-input-number v-model:value="editForm.rating" :min="0" :max="10" style="flex: 1" />
              <LockButton v-model:locked="editForm.rating_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="上映日期">
            <a-flex gap="middle" align="center">
              <a-input v-model:value="editForm.air_date" placeholder="YYYY-MM-DD" />
              <LockButton v-model:locked="editForm.air_date_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="首播日期">
            <a-flex gap="middle" align="center">
              <a-input v-model:value="firstAirDateValue" placeholder="YYYY-MM-DD" />
              <LockButton v-model:locked="editForm.first_air_date_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="末播日期">
            <a-flex gap="middle" align="center">
              <a-input v-model:value="lastAirDateValue" placeholder="YYYY-MM-DD" />
              <LockButton v-model:locked="editForm.last_air_date_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="内容分级">
            <a-flex gap="middle" align="center">
              <a-select v-model:value="editForm.content_rating"
                :options="(editForm.content_rating_opts || []).map(o => ({ label: o, value: o }))" allow-clear
                style="flex: 1" />
              <LockButton v-model:locked="editForm.content_rating_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="官方条目">
            <a-switch v-model:checked="editForm.is_official" />
          </a-form-item>
          <a-form-item label="海报类型">
            <a-input-number v-model:value="editForm.poster_type" :min="0" style="width: 200px" />
          </a-form-item>
          <a-form-item label="类型">
            <a-flex gap="middle" align="center">
              <a-select v-model:value="editForm.genres" mode="tags" :options="genreOptions"
                :field-names="{ label: 'value', value: 'id' }" option-filter-prop="value" placeholder="选择类型"
                :loading="genresLoading" allow-clear style="flex: 1" />
              <LockButton v-model:locked="editForm.genres_locked" />
            </a-flex>
          </a-form-item>
          <a-form-item label="制片地区">
            <a-flex gap="middle" align="center">
              <a-select v-model:value="editForm.production_countries" mode="multiple" :options="countryOptions"
                :field-names="{ label: 'value', value: 'key' }" placeholder="选择制片地区" :loading="countriesLoading"
                allow-clear style="flex: 1" />
              <LockButton v-model:locked="editForm.production_countries_locked" />
            </a-flex>
          </a-form-item>

          <a-divider orientation="left" plain>演职员</a-divider>
          <a-form-item>
            <LockButton v-model:locked="editForm.credits_locked">
              <span>锁定演职员</span>
            </LockButton>
          </a-form-item>
          <div v-if="editForm.credits && editForm.credits.length > 0">
            <div v-for="(credit, idx) in editForm.credits" :key="idx"
              style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;">
              <a-input v-model:value="credit.name" placeholder="姓名" style="width: 160px" />
              <a-select v-model:value="credit.job"
                :options="(editForm.job_types_opts || []).map(o => ({ label: o, value: o }))" allow-clear
                placeholder="职务" style="width: 160px" />
              <a-input v-model:value="credit.role" placeholder="角色" style="width: 200px" />
              <a-input-number v-model:value="credit.order" :min="1" placeholder="序" style="width: 80px" />
              <a-button danger size="small" @click="editForm.credits.splice(idx, 1)">删除</a-button>
            </div>
          </div>
          <a-button type="dashed" size="small" @click="addCredit">
            <template #icon>
              <PlusOutlined />
            </template>
            添加演职员
          </a-button>
        </a-form>
      </a-spin>
      <template #footer>
        <a-space>
          <a-button @click="editVisible = false">取消</a-button>
          <a-button :loading="searching" @click="handleSearch">
            <template #icon>
              <SearchOutlined />
            </template>
            搜索
          </a-button>
          <a-button type="primary" :loading="editSaving" @click="handleSaveEdit">保存</a-button>
        </a-space>
      </template>
    </a-modal>

    <!-- MetaTube 搜索结果弹窗 -->
    <a-modal v-model:open="searchVisible" width="800px" title="MetaTube 搜索结果" :footer="null"
      :body-style="{ maxHeight: '70vh', minHeight: '400px', overflow: 'auto' }">
      <a-spin :spinning="searching">
        <a-empty v-if="!searching && metaTubeStore.searchResults.length === 0" description="无搜索结果" />
        <a-row :gutter="[16, 16]">
          <a-col v-for="result in metaTubeStore.searchResults" :key="result.id + result.provider" :xs="24" :sm="12"
            :md="8">
            <a-card hoverable size="small" style="overflow: hidden" @click="handleSearchResultClick(result)">
              <template #cover>
                <img v-if="!uiStore.hideImages && (result.thumb_url || result.cover_url)"
                  :src="result.thumb_url || result.cover_url" style="height: 200px; width: 100%; object-fit: cover"
                  alt="cover" />
                <div v-else
                  style="height: 200px; display: flex; align-items: center; justify-content: center; background: #f5f5f5">
                  <span style="color: #999; font-size: 24px">{{ result.number?.charAt(0) || '?' }}</span>
                </div>
              </template>
              <a-card-meta>
                <template #title>
                  <div style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis"><a-tag
                      v-if="result.provider" color="blue">{{ result.provider }}</a-tag>{{ result.number }}</div>
                </template>
                <template #description>
                  <div style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-bottom: 4px">{{
                    result.title }}</div>
                  <a-space size="small" wrap>
                    <a-tag v-if="result.release_date">{{ formatDate(result.release_date) }}</a-tag>
                    <a-tag v-if="result.score" color="orange">评分 {{ result.score }}</a-tag>
                  </a-space>
                </template>
              </a-card-meta>
              <div v-if="result.actors && result.actors.length > 0"
                style="margin-top: 8px; font-size: 12px; color: #999">
                演员: {{ result.actors.join('、') }}
              </div>
            </a-card>
          </a-col>
        </a-row>
      </a-spin>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlayCircleOutlined, SearchOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons-vue'
import { useTrimMediaStore } from '@/stores/trimmedia'
import { useMetaTubeStore } from '@/stores/metatube'
import { useUiStore } from '@/stores/ui'
import LockButton from '@/components/LockButton.vue'
import { proxyImage } from '@/utils/image'
import { getEditDetail, saveEditDetail, getGenres, getCountries, batchCreateGenres, searchPersons, importPerson, type EditDetail, type EditCredit, type Genre, type Country, type PersonSearchResult } from '@/api/trimmedia'
import { type MovieSearchResult } from '@/api/metatube'
import type { MediaServerItem } from '@/api/trimmedia'

const route = useRoute()
const router = useRouter()
const store = useTrimMediaStore()
const metaTubeStore = useMetaTubeStore()
const uiStore = useUiStore()

const detailVisible = ref(false)
const gettingURL = ref(false)
const searchVisible = ref(false)
const searching = ref(false)

// 编辑相关
const editVisible = ref(false)
const editLoading = ref(false)
const editSaving = ref(false)
const editForm = ref<EditDetail | null>(null)
const genreOptions = ref<Genre[]>([])
const genresLoading = ref(false)
const countryOptions = ref<Country[]>([])
const countriesLoading = ref(false)

// first_air_date / last_air_date 可能为 null，用 computed 桥接到 input
const firstAirDateValue = computed({
  get: () => editForm.value?.first_air_date || '',
  set: (val: string) => { if (editForm.value) editForm.value.first_air_date = val || null },
})
const lastAirDateValue = computed({
  get: () => editForm.value?.last_air_date || '',
  set: (val: string) => { if (editForm.value) editForm.value.last_air_date = val || null },
})

const libraryId = computed(() => route.params.id as string)
const libraryName = computed(() => {
  const lib = store.libraries.find(l => l.id === libraryId.value)
  return lib?.name || libraryId.value
})

onMounted(async () => {
  // 如果 libraries 还没加载，先加载
  if (store.libraries.length === 0) {
    await store.fetchLibraries().catch(() => {
      message.warning('未连接到飞牛影视，请先配置')
    })
  }
  // 加载该媒体库的条目
  await store.fetchItems(libraryId.value, 0, 50).catch(() => {
    message.error('获取媒体列表失败')
  })
})

function backToLibraries() {
  router.push('/media')
}

function itemYear(item: MediaServerItem): string {
  const date = item.release_date || item.air_date || ''
  return date.slice(0, 4)
}

// 将 ISO 日期（如 2026-08-06T00:00:00Z）格式化为 YYYY-MM-DD
function formatDate(date: string): string {
  if (!date) return ''
  const d = new Date(date)
  if (isNaN(d.getTime())) return date
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

async function showDetail(item: MediaServerItem) {
  detailVisible.value = true
  await store.fetchItem(item.guid).catch(() => {
    message.error('获取详情失败')
  })
  await store.fetchPersons(item.guid).catch(() => { })
  if (store.currentItem?.type === 'TV') {
    await store.fetchSeasons(item.guid).catch(() => { })
  }
}

async function loadEpisodes(seasonId: string) {
  await store.fetchEpisodes(seasonId).catch(() => {
    message.error('获取剧集列表失败')
  })
}

async function handlePlay() {
  if (!store.currentItem) return
  gettingURL.value = true
  try {
    const url = await store.fetchPlayURL(store.currentItem.guid)
    window.open(url, '_blank')
  } catch {
    message.error('获取播放链接失败')
  } finally {
    gettingURL.value = false
  }
}

async function handleSearch() {
  if (!store.currentItem) return
  const keyword = store.currentItem.parent_title || store.currentItem.title
  if (!keyword) {
    message.warning('无法获取搜索关键词')
    return
  }
  metaTubeStore.searchResults = []
  searchVisible.value = true
  searching.value = true
  try {
    await metaTubeStore.searchMoviesData(keyword)
  } catch {
    message.error('MetaTube 搜索失败')
  } finally {
    searching.value = false
  }
}

async function loadGenres() {
  if (genreOptions.value.length > 0) return
  genresLoading.value = true
  try {
    const { data } = await getGenres()
    genreOptions.value = data || []
  } catch {
    // 类型列表加载失败不阻塞编辑
  } finally {
    genresLoading.value = false
  }
}

async function loadCountries() {
  if (countryOptions.value.length > 0) return
  countriesLoading.value = true
  try {
    const { data } = await getCountries()
    countryOptions.value = data || []
  } catch {
    // 地区列表加载失败不阻塞编辑
  } finally {
    countriesLoading.value = false
  }
}

async function handleEdit() {
  if (!store.currentItem) return
  editVisible.value = true
  editLoading.value = true
  // 并行加载编辑信息、类型列表与地区列表
  try {
    const [editRes] = await Promise.all([
      getEditDetail(store.currentItem.guid),
      loadGenres(),
      loadCountries(),
    ])
    editForm.value = editRes.data
  } catch {
    message.error('获取编辑信息失败')
    editVisible.value = false
  } finally {
    editLoading.value = false
  }
}

function addCredit() {
  if (!editForm.value) return
  if (!editForm.value.credits) {
    editForm.value.credits = []
  }
  const newCredit: EditCredit = {
    person_guid: '',
    name: '',
    job: '',
    role: '',
    order: editForm.value.credits.length + 1,
    profile_path: '',
  }
  editForm.value.credits.push(newCredit)
}

async function handleSaveEdit() {
  if (!editForm.value || !store.currentItem) return
  editSaving.value = true
  try {
    const pendingGenres = editForm.value.genres.filter(g => typeof g === 'string')
    const genres = editForm.value.genres.filter(g => typeof g === 'number')
    // 如果有待创建的新分类，先调用批量创建接口获取 id
    if (pendingGenres.length > 0) {
      try {
        const { data: newGenres } = await batchCreateGenres(pendingGenres)
        // 将新分类 id 合并入 genres
        for (const g of newGenres) {
          if (!genres.includes(g.id)) {
            genres.push(g.id)
          }
          // 同步更新下拉选项，避免显示空白
          if (!genreOptions.value.some(opt => opt.id === g.id)) {
            genreOptions.value.push(g)
          }
        }
        editForm.value.genres = genres
      } catch {
        message.warning('部分新分类创建失败，将仅保存已存在的分类')
      }
    }
    await saveEditDetail(store.currentItem.guid, editForm.value)
    message.success('保存成功')
    editVisible.value = false
  } catch {
    message.error('保存失败')
  } finally {
    editSaving.value = false
  }
}

// 点击搜索结果卡片：调用影片详情接口并填入编辑弹窗
async function handleSearchResultClick(result: MovieSearchResult) {
  if (!editForm.value) return
  searching.value = true
  try {
    const info = await metaTubeStore.fetchMovieDetail(result.provider, result.id)
    if (!info) {
      message.warning('未获取到影片详情')
      return
    }
    // 将 MetaTube 影片信息填入编辑表单（仅覆盖未锁定字段）
    if (!editForm.value.title_locked && info.title) {
      editForm.value.title = info.title
    }
    if (!editForm.value.overview_locked && info.summary) {
      editForm.value.overview = info.summary
    }
    if (!editForm.value.rating_locked && info.score) {
      editForm.value.rating = info.score
    }
    if (!editForm.value.content_rating_locked) {
      editForm.value.content_rating = 'JP-18+'
    }
    if (!editForm.value.air_date_locked && info.release_date) {
      editForm.value.air_date = formatDate(info.release_date)
    }
    if (!editForm.value.posters_locked && (info.cover_url || info.big_cover_url || info.thumb_url)) {
      editForm.value.posters = info.big_cover_url || info.cover_url || info.thumb_url
    }
    if (!editForm.value.backdrops_locked && (info.thumb_url || info.big_thumb_url)) {
      editForm.value.backdrops = info.big_thumb_url || info.thumb_url
    }
    if (!editForm.value.genres_locked && info.genres && info.genres.length > 0) {
      const matchedIds: number[] = []
      const unmatchedTexts: string[] = []
      for (const g of info.genres) {
        const found = genreOptions.value.find(opt => opt.value === g)
        if (found) {
          matchedIds.push(found.id)
        } else {
          unmatchedTexts.push(g)
        }
      }
      editForm.value.genres = [...matchedIds, ...unmatchedTexts]
    }
    // 演职员：actors 为字符串数组，通过飞牛演员搜索接口查询是否已存在
    if (!editForm.value.credits_locked && info.actors && info.actors.length > 0) {
      const credits: EditCredit[] = []
      for (let idx = 0; idx < info.actors.length; idx++) {
        const name = info.actors[idx]
        try {
          const { data: persons } = await searchPersons(name, 1, 10)
          if (persons && persons.length > 0) {
            // 已存在，使用第一个匹配的演员信息
            const person = persons[0]
            credits.push({
              person_guid: person.guid,
              name: person.name,
              job: 'Actor',
              role: '',
              order: idx + 1,
              profile_path: person.profile,
            })
          } else {
            // 不存在，通过 MetaTube 搜索演员 → 下载图片 → 上传 → 创建
            try {
              const { data: imported } = await importPerson(name)
              credits.push({
                person_guid: imported.guid,
                name: imported.name,
                job: 'Actor',
                role: '',
                order: idx + 1,
                profile_path: imported.profile_path,
              })
            } catch {
              // 导入失败，使用空信息
              credits.push({
                person_guid: '',
                name,
                job: 'Actor',
                role: '',
                order: idx + 1,
                profile_path: '',
              })
            }
          }
        } catch {
          // 搜索失败时仍使用当前添加方法
          credits.push({
            person_guid: '',
            name,
            job: 'Actor',
            role: '',
            order: idx + 1,
            profile_path: '',
          })
        }
      }
      editForm.value.credits = credits
    }
    message.success('已填入影片信息')
    searchVisible.value = false
  } catch {
    message.error('获取影片详情失败')
  } finally {
    searching.value = false
  }
}
</script>

<style scoped>
:deep(.ant-col) {
  min-width: 0;
}

:deep(.ant-card-body) {
  overflow: hidden;
}

:deep(.ant-card-meta) {
  min-width: 0;
}

:deep(.ant-card-meta-detail) {
  min-width: 0;
  overflow: hidden;
}
</style>