<template>
  <!-- MetaTube 搜索结果弹窗 -->
  <a-modal v-model:open="visible" width="800px" title="MetaTube 搜索结果" :footer="null"
    :body-style="{ maxHeight: '70vh', minHeight: '400px', overflow: 'auto' }">
    <a-spin :spinning="searching" style="min-height: 400px">
      <a-empty v-if="!searching && metaTubeStore.searchResults.length === 0" description="无搜索结果" />
      <a-row :gutter="[16, 16]">
        <a-col v-for="result in metaTubeStore.searchResults" :key="result.id + result.provider" :xs="24" :sm="12"
          :md="8">
          <a-card hoverable size="small" style="overflow: hidden" @click="handleResultClick(result)">
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
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { useMetaTubeStore } from '@/stores/metatube'
import { useUiStore } from '@/stores/ui'
import { type MovieSearchResult, type MovieInfo } from '@/api/metatube'

const emit = defineEmits<{
  /** 选中搜索结果并获取到影片详情后触发 */
  select: [info: MovieInfo]
}>()

const metaTubeStore = useMetaTubeStore()
const uiStore = useUiStore()

const visible = ref(false)
const searching = ref(false)
const keyword = ref('')

function open(val: string) {
  keyword.value = val
  visible.value = true
  doSearch()
}

defineExpose({
  open,
})

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

async function doSearch() {
  if (!keyword.value) {
    message.warning('无法获取搜索关键词')
    return
  }
  metaTubeStore.searchResults = []
  searching.value = true
  try {
    await metaTubeStore.searchMoviesData(keyword.value)
  } catch {
    message.error('MetaTube 搜索失败')
  } finally {
    searching.value = false
  }
}

// 点击搜索结果卡片：调用影片详情接口并通知父组件
async function handleResultClick(result: MovieSearchResult) {
  searching.value = true
  try {
    const info = await metaTubeStore.fetchMovieDetail(result.provider, result.id)
    if (!info) {
      message.warning('未获取到影片详情')
      return
    }
    emit('select', info)
    visible.value = false
  } catch {
    message.error('获取影片详情失败')
  } finally {
    searching.value = false
  }
}
</script>
