<template>
  <div class="settings-page">
    <a-card title="系统设置">
      <div class="config-links">
        <a-button size="large" block @click="router.push('/config')">
          <template #icon><SettingOutlined /></template>
          飞牛影视设置
        </a-button>
        <a-button size="large" block @click="router.push('/metatube-config')">
          <template #icon><TranslationOutlined /></template>
          MetaTube设置
        </a-button>
        <a-button size="large" block @click="router.push('/guangya-config')">
          <template #icon><CloudOutlined /></template>
          光鸭云盘设置
        </a-button>
        <a-button size="large" block @click="router.push('/smartstrm-config')">
          <template #icon><LinkOutlined /></template>
          SmartStrm设置
        </a-button>
        <a-button size="large" block @click="router.push('/storage-config')">
          <template #icon><DatabaseOutlined /></template>
          存储设置
        </a-button>
        <a-button size="large" block @click="router.push('/guangya-cloud-jobs')">
          <template #icon><HistoryOutlined /></template>
          云添加记录
        </a-button>
      </div>
    </a-card>
    <a-card title="显示设置">
      <div class="setting-row">
        <span class="setting-label"><PictureOutlined /> 图片显隐</span>
        <a-switch v-model:checked="hideImagesChecked" checked-children="隐图" un-checked-children="显图"
          aria-label="隐藏图片" />
      </div>
      <div class="setting-row">
        <span class="setting-label"><DesktopOutlined /> 主题模式</span>
        <a-segmented v-model:value="themeMode" :options="themeModeOptions" aria-label="主题模式" />
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { SettingOutlined, TranslationOutlined, CloudOutlined, LinkOutlined, DatabaseOutlined, HistoryOutlined, PictureOutlined, DesktopOutlined } from '@ant-design/icons-vue'
import { useUiStore, type ThemeMode } from '@/stores/ui'

const router = useRouter()
const uiStore = useUiStore()

const hideImagesChecked = computed({
  get: () => uiStore.hideImages,
  set: (val: boolean) => uiStore.setHideImages(val),
})

const themeModeOptions = [
  { label: '跟随系统', value: 'system' },
  { label: '深色模式', value: 'dark' },
  { label: '亮色模式', value: 'light' },
]

const themeMode = computed({
  get: () => uiStore.themeMode,
  set: (val: ThemeMode) => uiStore.setThemeMode(val),
})
</script>

<style scoped lang="scss">
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.config-links {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;

  & + & {
    margin-top: 20px;
  }
}

.setting-label {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
