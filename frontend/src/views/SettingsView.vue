<template>
  <div class="settings-page">
    <a-card title="配置">
      <div class="config-links">
        <a-button size="large" block @click="router.push('/config')">
          <template #icon><SettingOutlined /></template>
          飞牛影视配置
        </a-button>
        <a-button size="large" block @click="router.push('/metatube-config')">
          <template #icon><TranslationOutlined /></template>
          MetaTube 配置
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
        <span class="setting-label">
          <component :is="uiStore.darkMode ? BulbFilled : BulbOutlined" /> 深色模式
        </span>
        <a-switch v-model:checked="darkModeChecked" checked-children="开" un-checked-children="关"
          aria-label="深色模式" />
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { SettingOutlined, TranslationOutlined, PictureOutlined, BulbFilled, BulbOutlined } from '@ant-design/icons-vue'
import { useUiStore } from '@/stores/ui'

const router = useRouter()
const uiStore = useUiStore()

const hideImagesChecked = computed({
  get: () => uiStore.hideImages,
  set: (val: boolean) => uiStore.setHideImages(val),
})

const darkModeChecked = computed({
  get: () => uiStore.darkMode,
  set: (val: boolean) => uiStore.setDarkMode(val),
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
