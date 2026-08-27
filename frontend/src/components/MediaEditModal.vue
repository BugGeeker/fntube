<template>
  <!-- 编辑弹窗 -->
  <a-modal v-model:open="editVisible" width="900px" title="编辑媒体信息"
    :body-style="{ maxHeight: '80vh', minHeight: '600px', overflow: 'auto' }">
    <a-spin :spinning="editLoading" style="min-height: 600px; width: 100%">
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
            <a-spin v-if="translatingTitle" size="small" />
            <LockButton v-model:locked="editForm.title_locked" />
          </a-flex>
        </a-form-item>

        <a-form-item label="简介">
          <a-flex gap="middle" align="center">
            <a-textarea v-model:value="editForm.overview" :rows="4" />
            <a-spin v-if="translatingOverview" size="small" />
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
        <a-button @click="handleSearch">
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
  <MetaTubeSearchModal ref="searchModal" @select="handleSearchSelect" />
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import { SearchOutlined, PlusOutlined } from '@ant-design/icons-vue'
import { useMetaTubeStore } from '@/stores/metatube'
import LockButton from '@/components/LockButton.vue'
import MetaTubeSearchModal from '@/components/MetaTubeSearchModal.vue'
import {
  getEditDetail,
  saveEditDetail,
  getGenres,
  getCountries,
  batchCreateGenres,
  searchPersons,
  getStreamList,
  importPerson,
  downloadAndUploadImage,
  type EditDetail,
  type EditCredit,
  type Genre,
  type Country,
  type MediaItem,
} from '@/api/trimmedia'
import { translateText, type MetaTubeConfig, type MovieInfo } from '@/api/metatube'
import { formatDateOnly } from '@/utils/format'

const emit = defineEmits<{
  saved: []
}>()

const metaTubeStore = useMetaTubeStore()
const searchModal = ref<typeof MetaTubeSearchModal>()

//// 编辑相关
const editVisible = ref(false)
const editLoading = ref(false)
const editSaving = ref(false)
const editForm = ref<EditDetail | null>(null)
const currentItem = ref<MediaItem | null>(null)
const genreOptions = ref<Genre[]>([])
const genresLoading = ref(false)
const countryOptions = ref<Country[]>([])
const countriesLoading = ref(false)

// 翻译状态
const translatingTitle = ref(false)
const translatingOverview = ref(false)

// first_air_date / last_air_date 可能为 null，用 computed 桥接到 input
const firstAirDateValue = computed({
  get: () => editForm.value?.first_air_date || '',
  set: (val: string) => { if (editForm.value) editForm.value.first_air_date = val || null },
})
const lastAirDateValue = computed({
  get: () => editForm.value?.last_air_date || '',
  set: (val: string) => { if (editForm.value) editForm.value.last_air_date = val || null },
})

// 将 ISO 日期（如 2026-08-06T00:00:00Z）格式化为 YYYY-MM-DD
const open = async (item: MediaItem) => {
  currentItem.value = item
  editVisible.value = true
  editLoading.value = true
  // 并行加载编辑信息、类型列表与地区列表
  try {
    const [editRes] = await Promise.all([
      getEditDetail(item.guid),
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

async function openWithMovieInfo(item: MediaItem, info: MovieInfo) {
  await open(item)
  await fillEditFormFromMovieInfo(info)
}

defineExpose({
  open,
  openWithMovieInfo,
})

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
  if (!editForm.value || !currentItem.value) return
  editSaving.value = true
  try {
    await saveEditFormToTrim()
    message.success('保存成功')
    editVisible.value = false
    emit('saved')
  } catch {
    message.error('保存失败')
  } finally {
    editSaving.value = false
  }
}

async function handleSearch() {
  if (!currentItem.value) return
  try {
    const { data } = await getStreamList(currentItem.value.guid)
    const fileName = data.files[0]?.file_name
    const keyword = fileName?.replace(/\.[^.]+$/, '')
    if (!keyword) {
      message.warning('无法获取媒体文件名')
      return
    }
    searchModal.value?.open(keyword)
  } catch {
    message.error('获取媒体文件信息失败')
  }
}

// 选中搜索结果后，将影片详情填入编辑弹窗
async function handleSearchSelect(info: MovieInfo) {
  if (!editForm.value) return
  try {
    await fillEditFormFromMovieInfo(info)
    message.success('已填入影片信息')
  } catch {
    message.error('填入影片信息失败')
  }
}

// 将 MetaTube 影片信息填入编辑表单的核心逻辑（抽取以便复用）
async function fillEditFormFromMovieInfo(info: MovieInfo) {
  if (!editForm.value) return
  // 将 MetaTube 影片信息填入编辑表单（仅覆盖未锁定字段）
  if (!editForm.value.title_locked && info.number) {
    editForm.value.title = info.number + (info.title ? (' ' + info.title) : '')
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
    editForm.value.air_date = formatDateOnly(info.release_date)
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

  // 翻译逻辑：根据配置的 translate_mode 异步翻译标题和/或简介
  const cfg = metaTubeStore.config as MetaTubeConfig | null
  const mode = cfg?.translate_mode || 'none'
  if (mode !== 'none') {
    const shouldTranslateTitle = (mode === 'title' || mode === 'title_and_summary')
    const shouldTranslateOverview = (mode === 'summary' || mode === 'title_and_summary')
    const translateTasks: Promise<void>[] = []
    if (shouldTranslateTitle && !editForm.value.title_locked && info.title) {
      translatingTitle.value = true
      translateTasks.push(
        translateText(info.title).then(({ data: res }) => {
          if (res?.data?.translated_text && editForm.value && !editForm.value.title_locked) {
            editForm.value.title = info.number + ' ' + res.data.translated_text
          }
        }).catch(() => {
          message.warning('标题翻译失败')
        }).finally(() => {
          translatingTitle.value = false
        })
      )
    }
    if (shouldTranslateOverview && !editForm.value.overview_locked && info.summary) {
      translatingOverview.value = true
      translateTasks.push(
        translateText(info.summary).then(({ data: res }) => {
          if (res?.data?.translated_text && editForm.value && !editForm.value.overview_locked) {
            editForm.value.overview = res.data.translated_text
          }
        }).catch(() => {
          message.warning('简介翻译失败')
        }).finally(() => {
          translatingOverview.value = false
        })
      )
    }
    await Promise.all(translateTasks)
  }
}

// 保存编辑信息到飞牛
async function saveEditFormToTrim(): Promise<boolean> {
  if (!editForm.value || !currentItem.value) return false
  // 封面/背景图为 http 网络图片时，先下载并上传到飞牛，再使用飞牛返回的本地路径保存
  const imageFields: Array<{ key: 'posters' | 'backdrops'; type: string }> = [
    { key: 'posters', type: 'poster' },
    { key: 'backdrops', type: 'backdrop' },
  ]
  for (const { key, type } of imageFields) {
    const value = editForm.value[key]
    if (typeof value === 'string' && (value.startsWith('http://') || value.startsWith('https://'))) {
      try {
        const { data } = await downloadAndUploadImage(value, type)
        editForm.value[key] = data.path
      } catch {
        message.warning('网络图片上传飞牛失败，将保留原地址')
      }
    }
  }
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
  await saveEditDetail(currentItem.value.guid, editForm.value)
  return true
}
</script>
