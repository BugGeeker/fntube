<template>
  <div>
    <a-card title="媒体刮削检查">
      <template #extra>
        <a-space>
          <a-select
            v-model:value="selectedLibraryId"
            placeholder="选择媒体库"
            :options="libraryOptions"
            :loading="librariesLoading"
            class="library-select"
            allow-clear
          />
          <a-button type="primary" :loading="checking" :disabled="!selectedLibraryId" @click="checkInconsistentItems">
            查询
          </a-button>
        </a-space>
      </template>

      <a-alert
        v-if="selectedLibraryName && hasQueried"
        :message="`${selectedLibraryName}：共发现 ${inconsistentItems.length} 个文件名不一致的媒体`"
        type="info"
        show-icon
        class="result-summary"
      />

      <a-spin :spinning="checking">
        <a-empty
          v-if="!checking && !hasQueried"
          description="请选择媒体库后查询"
        />
        <a-empty
          v-else-if="!checking && inconsistentItems.length === 0"
          description="未发现文件名不一致的媒体"
        />
        <a-list v-else :data-source="inconsistentItems" bordered>
          <template #renderItem="{ item }">
            <a-list-item>
              <template #extra>
                <a-button
                  size="small"
                  :loading="rescrapingGuid === item.guid"
                  @click="handleRescrape(item.guid)"
                >
                  重新刮削
                </a-button>
              </template>
              <a-list-item-meta :title="item.title || item.original_title || item.guid">
                <template #description>
                  <div class="inconsistent-detail">
                    <span>文件名：{{ item.file_name }}</span>
                    <span>文件番号：{{ item.expected_number }}</span>
                    <span>刮削番号：{{ item.actual_number || '未识别' }}</span>
                  </div>
                  <div class="inconsistent-reason">{{ item.reason }}</div>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
      </a-spin>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { getInconsistentItems, getLibraries, type InconsistentMediaItem, type Library } from '@/api/trimmedia'
import { rescrapeItem } from '@/api/scrapelog'

const libraries = ref<Library[]>([])
const librariesLoading = ref(false)
const selectedLibraryId = ref<string>()
const inconsistentItems = ref<InconsistentMediaItem[]>([])
const checking = ref(false)
const hasQueried = ref(false)
const rescrapingGuid = ref<string | null>(null)

const libraryOptions = computed(() =>
  libraries.value.map(library => ({
    label: library.name,
    value: library.id,
  })),
)

const selectedLibraryName = computed(() =>
  libraries.value.find(library => library.id === selectedLibraryId.value)?.name || '',
)

onMounted(async () => {
  librariesLoading.value = true
  try {
    const { data } = await getLibraries()
    libraries.value = data || []
  } catch {
    message.warning('未连接到飞牛影视，请先配置')
  } finally {
    librariesLoading.value = false
  }
})

async function checkInconsistentItems() {
  if (!selectedLibraryId.value) return
  checking.value = true
  hasQueried.value = true
  inconsistentItems.value = []
  try {
    const { data } = await getInconsistentItems(selectedLibraryId.value)
    inconsistentItems.value = data.items || []
  } catch {
    message.error('查询文件名不一致媒体失败')
  } finally {
    checking.value = false
  }
}

async function handleRescrape(guid: string) {
  rescrapingGuid.value = guid
  try {
    const { data } = await rescrapeItem(guid)
    inconsistentItems.value = inconsistentItems.value.filter(item => item.guid !== guid)
    message.success(data.message || '重新刮削已开始')
  } catch {
    message.error('重新刮削失败')
  } finally {
    rescrapingGuid.value = null
  }
}
</script>

<style scoped>
.library-select {
  width: 220px;
}

.result-summary {
  margin-bottom: 16px;
}

.inconsistent-detail {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
}

.inconsistent-reason {
  margin-top: 6px;
  color: #ff4d4f;
}

@media (max-width: 639px) {
  .library-select {
    width: 160px;
  }
}
</style>
