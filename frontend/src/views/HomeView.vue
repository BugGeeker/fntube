<template>
  <div class="dashboard">
    <!-- 统计区 -->
    <a-row :gutter="[16, 16]" class="stats-row">
      <a-col :xs="12" :sm="12" :md="6">
        <a-card>
          <a-statistic title="媒体数量" :value="summary?.total_media ?? 0" :loading="loading">
            <template #prefix><VideoCameraOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="12" :md="6">
        <a-card>
          <a-statistic title="本周新增" :value="summary?.weekly_new_media ?? 0" :loading="loading">
            <template #prefix><RiseOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="12" :md="6">
        <a-card>
          <a-statistic title="累计刮削" :value="summary?.total_scrapes ?? 0" :loading="loading">
            <template #prefix><FileSearchOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="12" :md="6">
        <a-card>
          <a-statistic title="本周刮削" :value="summary?.weekly_scrapes ?? 0" :loading="loading">
            <template #prefix><ScheduleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 刮削区 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :md="14">
        <a-card title="近15天刮削趋势">
          <v-chart class="chart" :option="chartOption" autoresize />
        </a-card>
      </a-col>
      <a-col :xs="24" :md="10">
        <a-card title="刮削计划任务">
          <a-spin :spinning="loading">
            <a-list :data-source="summary?.task_summary || []" size="small">
              <template #renderItem="{ item }">
                <a-list-item>
                  <a-list-item-meta :title="item.name" :description="item.last_run_at ? formatDateTime(item.last_run_at) : '-'">
                    <template #avatar>
                      <a-badge :status="item.is_running ? 'processing' : (item.enabled ? 'success' : 'default')" />
                    </template>
                  </a-list-item-meta>
                  <template #actions>
                    <span v-if="item.is_running" style="color: #1677ff">运行中</span>
                    <span v-else-if="item.enabled" style="color: #52c41a">已启用</span>
                    <span v-else style="color: #999">已停用</span>
                  </template>
                </a-list-item>
              </template>
              <template #footer>
                <span v-if="!summary?.task_summary?.length">暂无刮削计划任务</span>
              </template>
            </a-list>
          </a-spin>
        </a-card>
      </a-col>
    </a-row>

    <!-- 最近入库 -->
    <a-card title="最近入库" style="margin-top: 16px">
      <a-spin :spinning="latestLoading">
        <a-row :gutter="[12, 12]">
          <a-col
            v-for="item in latestItems"
            :key="item.id"
            :xs="24" :sm="12" :md="12" :lg="8" :xl="6" :xxl="4"
          >
            <div class="media-card" @click="openItem(item)">
              <MediaImage
                :src="item.image"
                :alt="item.title"
                ratio="3 / 2"
              />
              <div class="media-overlay">
                <span class="media-title">{{ item.title }}</span>
              </div>
            </div>
          </a-col>
        </a-row>
        <a-empty v-if="!latestLoading && latestItems.length === 0" description="暂无入库媒体" />
      </a-spin>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  VideoCameraOutlined,
  RiseOutlined,
  FileSearchOutlined,
  ScheduleOutlined,
} from '@ant-design/icons-vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { getDashboardSummary, type DashboardSummary } from '@/api/dashboard'
import { getLatest, type PlayItem } from '@/api/trimmedia'
import { formatDateTime } from '@/utils/format'
import MediaImage from '@/components/MediaImage.vue'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent])

const loading = ref(false)
const latestLoading = ref(false)
const summary = ref<DashboardSummary | null>(null)
const latestItems = ref<PlayItem[]>([])

const chartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: 30, right: 16, top: 20, bottom: 28 },
  xAxis: {
    type: 'category',
    data: summary.value?.daily_scrapes.map((d) => d.date) || [],
    axisLabel: { fontSize: 11 },
  },
  yAxis: { type: 'value', minInterval: 1 },
  series: [
    {
      data: summary.value?.daily_scrapes.map((d) => d.count) || [],
      type: 'line',
      smooth: true,
      areaStyle: { opacity: 0.15 },
      lineStyle: { width: 2 },
      itemStyle: { color: '#1677ff' },
    },
  ],
}))

function openItem(item: PlayItem) {
  if (item.id) {
    window.open(`#/media/${item.id}`, '_self')
  }
}

onMounted(async () => {
  loading.value = true
  latestLoading.value = true
  try {
    const [summaryResp, latestResp] = await Promise.all([
      getDashboardSummary(),
      getLatest(20),
    ])
    summary.value = summaryResp.data
    latestItems.value = latestResp.data || []
  } catch {
    // 忽略错误
  } finally {
    loading.value = false
    latestLoading.value = false
  }
})
</script>

<style scoped>
.dashboard {
  max-width: 1400px;
}

.chart {
  height: 280px;
}

.stats-row :deep(.ant-statistic-content) {
  font-size: 28px;
}

/* 媒体卡片 */
.media-card {
  position: relative;
  width: 100%;
  aspect-ratio: 3 / 2;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  background: #f0f0f0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  transition: box-shadow 0.3s;
}

.media-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.18);
}

.media-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: flex-end;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.75) 0%, rgba(0, 0, 0, 0) 60%);
  opacity: 0;
  transition: opacity 0.3s ease;
  padding: 8px;
}

.media-card:hover .media-overlay {
  opacity: 1;
}

.media-title {
  color: #fff;
  font-size: 13px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
