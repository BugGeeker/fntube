<template>
  <div>
    <!-- 媒体条目列表 -->
    <a-card :title="libraryName || '加载中...'">
      <template #extra>
        <a-space>
          <a-button @click="backToLibraries">返回媒体库</a-button>
          <a-button @click="handleRefresh">
            <template #icon>
              <ReloadOutlined />
            </template>
          </a-button>
        </a-space>
      </template>
      <a-spin :spinning="store.loading">
        <div class="media-item-grid">
          <div v-for="item in store.items" :key="item.guid" class="media-item-cell">
            <a-card hoverable size="small" @click="showDetail(item)" class="media-card">
              <template #cover>
                <div style="position: relative;">
                  <MediaImage :src="item.poster" :alt="item.title" :ratio="posterRatio" />

                  <div class="card-actions" @click.stop>
                    <a-button size="medium" shape="circle" @click.stop="handleCardEdit(item)" title="播放">
                      <template #icon>
                        <CaretRightOutlined />
                      </template>
                    </a-button>
                    <a-button size="medium" shape="circle" @click.stop="handleCardEdit(item)" title="编辑">
                      <template #icon>
                        <EditOutlined />
                      </template>
                    </a-button>
                    <a-button size="medium" shape="circle" :loading="scrapingItem === item.guid"
                      @click.stop="handleScrape(item)" title="刮削">
                      <template #icon>
                        <ThunderboltOutlined />
                      </template>
                    </a-button>
                  </div>
                </div>
              </template>
              <a-card-meta>
                <template #title>
                  <div style="overflow-wrap: anywhere; word-break: break-word">{{ item.title }}</div>
                </template>
                <template #description>
                  <span>{{ itemYear(item) || '未知年份' }}</span>
                  <a-tag v-if="item.type === 'TV'" color="blue" style="margin-left: 8px">剧集</a-tag>
                  <a-tag v-else-if="item.type === 'Movie'" color="orange" style="margin-left: 8px">电影</a-tag>
                </template>
              </a-card-meta>
            </a-card>
          </div>
        </div>
        <a-empty v-if="!store.loading && store.items.length === 0" description="该媒体库暂无内容" />
      </a-spin>
      <div v-if="!store.loading && store.itemsTotal > 0"
        style="display: flex; justify-content: center; margin-top: 16px">
        <a-pagination v-model:current="currentPage" v-model:page-size="pageSize" :total="store.itemsTotal"
          :page-size-options="[10, 20, 50, 100]" show-size-changer :show-total="(total: number) => `共 ${total} 项`"
          @change="handlePageChange" @show-size-change="handlePageChange" />
      </div>
    </a-card>

    <!-- 媒体详情弹窗 -->
    <MediaDetailModal ref="mediaDetailModelRef" :item="store.currentItem" @edit="handleEdit" />

    <!-- 编辑弹窗 -->
    <MediaEditModal ref="editModalRef" />

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { EditOutlined, ThunderboltOutlined, CaretRightOutlined } from '@ant-design/icons-vue'
import { useTrimMediaStore } from '@/stores/trimmedia'
import { useMetaTubeStore } from '@/stores/metatube'
import MediaDetailModal from '@/components/MediaDetailModal.vue'
import MediaEditModal from '@/components/MediaEditModal.vue'
import { rescrapeItem } from '@/api/scrapelog'
import MediaImage from '@/components/MediaImage.vue'
import { getLibraryViewType, type MediaItem } from '@/api/trimmedia'
import { ReloadOutlined } from '@ant-design/icons-vue'

const route = useRoute()
const router = useRouter()
const store = useTrimMediaStore()
const metaTubeStore = useMetaTubeStore()

const mediaDetailModelRef = ref<typeof MediaDetailModal>()
const editModalRef = ref<typeof MediaEditModal>()

// 刮削状态
const scrapingItem = ref<string | null>(null)

const libraryId = computed(() => route.params.id as string)
const libraryName = computed(() => {
  const lib = store.libraries.find(l => l.id === libraryId.value)
  return lib?.name || libraryId.value
})
const viewType = ref('')
const posterRatio = computed(() => viewType.value === 'horizontal_poster' ? '3 / 2' : '2 / 3')

// 分页
const currentPage = ref(1)
const pageSize = ref(20)

async function loadItems() {
  try {
    await store.fetchItems(libraryId.value, currentPage.value - 1, pageSize.value)
  } catch {
    message.error('获取媒体列表失败')
  }
}

function handlePageChange(page: number, size: number) {
  currentPage.value = page
  if (size) pageSize.value = size
  loadItems()
}

onMounted(async () => {
  // 如果 libraries 还没加载，先加载
  if (store.libraries.length === 0) {
    await store.fetchLibraries().catch(() => {
      message.warning('未连接到飞牛影视，请先配置')
    })
  }
  // 加载 MetaTube 配置（用于判断翻译模式）
  await metaTubeStore.fetchConfig().catch(() => { })
  const { data } = await getLibraryViewType(libraryId.value).catch(() => ({ data: { view_type: '' } }))
  viewType.value = data.view_type
  // 加载该媒体库第一页条目
  currentPage.value = 1
  await loadItems()
})

// 刷新列表
function handleRefresh() {
  loadItems()
}

function backToLibraries() {
  router.push('/media')
}

function itemYear(item: MediaItem): string {
  const date = item.release_date || item.air_date || ''
  return date.slice(0, 4)
}

async function showDetail(item: MediaItem) {
  mediaDetailModelRef.value?.open(item.guid)
}

async function handleEdit(item?: MediaItem) {
  editModalRef.value?.open(item)
}

// 卡片编辑按钮
function handleCardEdit(item: MediaItem) {
  handleEdit(item)
}

async function handleScrape(item: MediaItem) {
  scrapingItem.value = item.guid
  try {
    const { data } = await rescrapeItem(item.guid)
    message.success(data.message || '刮削已开始')
    // 延迟刷新列表
    setTimeout(() => loadItems(), 5000)
  } catch {
    message.error('刮削失败')
  } finally {
    scrapingItem.value = null
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

.media-item-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.media-item-cell :deep(.ant-card) {
  width: 100%;
}

.media-item-cell {
  flex: 1 1 clamp(180px, 20vw, 250px);
  min-width: 180px;
  max-width: 250px;
}

.media-card {
  position: relative;

  &:hover {
    .card-actions {
      opacity: 1;
    }
  }
}

/* 卡片操作按钮：hover 时显示 */
.card-actions {
  position: absolute;
  bottom: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s ease;
}
</style>