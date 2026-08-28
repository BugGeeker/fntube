<template>
  <!-- MetaTube 搜索结果弹窗 -->
  <a-modal v-model:open="visible" width="800px" title="MetaTube 搜索结果" :footer="null"
    :body-style="{ maxHeight: '70vh', minHeight: '400px', overflow: 'auto' }">
    <a-space-compact style="width: 100%; margin-bottom: 16px">
      <a-input v-model:value="keyword" placeholder="输入搜索关键词" @press-enter="doSearch" />
      <a-button type="primary" :loading="searching" @click="doSearch">搜索</a-button>
    </a-space-compact>
    <a-spin :spinning="searching" style="min-height: 400px">
      <a-empty v-if="!searching && metaTubeStore.searchResults.length === 0" description="无搜索结果" />
      <a-row :gutter="[16, 16]">
        <a-col v-for="result in metaTubeStore.searchResults" :key="result.id + result.provider" :xs="12" :sm="8" :md="6" :lg="4" :xl="3" :xxl="2" >
          <a-card hoverable size="small" style="overflow: hidden" @click="handleResultClick(result)">
            <template #cover>
              <MediaImage :src="result.thumb_url || result.cover_url" alt="cover" :fallback="result.number"
                ratio="2/3" />

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
                  <a-tag v-if="result.release_date">{{ formatDateOnly(result.release_date) }}</a-tag>
                  <a-tag v-if="result.score" color="orange">评分 {{ result.score }}</a-tag>
                </a-space>
              </template>
            </a-card-meta>
            <div v-if="result.actors && result.actors.length > 0" style="margin-top: 8px; font-size: 12px; color: #999">
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
import { type MovieSearchResult, type MovieInfo } from '@/api/metatube'
import { formatDateOnly } from '@/utils/format'
import MediaImage from './MediaImage.vue'

const emit = defineEmits<{
  /** 选中搜索结果并获取到影片详情后触发 */
  select: [info: MovieInfo]
}>()

const metaTubeStore = useMetaTubeStore()

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
