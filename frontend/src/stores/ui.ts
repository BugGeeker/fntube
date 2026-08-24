import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const STORAGE_KEY = 'fntube_hide_images'

/**
 * 全局 UI 相关状态，目前主要用于「隐藏图片」开关。
 *
 * 该状态持久化到 localStorage，刷新页面后仍生效；
 * 同时在 window 上派发 `fntube:hide-images-change` 事件，
 * 便于未使用 Pinia 的工具函数（如 proxyImage）感知变化。
 */
export const useUiStore = defineStore('ui', () => {
  const stored = localStorage.getItem(STORAGE_KEY)
  const hideImages = ref(stored === 'true')

  watch(
    hideImages,
    (val) => {
      localStorage.setItem(STORAGE_KEY, String(val))
      window.dispatchEvent(new CustomEvent('fntube:hide-images-change', { detail: val }))
    }
  )

  function setHideImages(val: boolean) {
    hideImages.value = val
  }

  function toggleHideImages() {
    hideImages.value = !hideImages.value
  }

  return {
    hideImages,
    setHideImages,
    toggleHideImages,
  }
})
