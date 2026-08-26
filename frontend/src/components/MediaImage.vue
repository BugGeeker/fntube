<template>
  <a-image
    v-if="src && !uiStore.hideImages"
    :src="proxyImage(src)"
    class="media-image"
    :alt="alt"
    :ratio="ratio"
  />
  <div v-else class="media-image media-image-placeholder">
    <span class="media-image-placeholder-text">{{ fallbackText }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'
import { proxyImage } from '@/utils/image'

const props = withDefaults(defineProps<{
  src?: string | null
  alt?: string
  ratio?: string
  fallback?: string
  fit?: string
  placeholderBg?: string
  placeholderColor?: string
  placeholderFontSize?: string
  borderRadius?: string
}>(), {
  alt: 'poster',
  ratio: '2 / 3',
  fallback: '',
  fit: 'cover',
  placeholderBg: '#f0f0f0',
  placeholderColor: '#999',
  placeholderFontSize: '32px',
  borderRadius: '8px',
})

const uiStore = useUiStore()

const fallbackText = computed(() => {
  if (props.fallback) return props.fallback
  return (props.alt || '?').split(' ')[0]
})
</script>

<style scoped>
.media-image {
  width: 100%;
  aspect-ratio: v-bind('props.ratio');
  object-fit: v-bind('props.fit');
  display: block;
  border-radius: v-bind('props.borderRadius');
}

.media-image-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: v-bind('props.placeholderBg');
}

.media-image-placeholder-text {
  padding: 0 12px;
  color: v-bind('props.placeholderColor');
  font-size: v-bind('props.placeholderFontSize');
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
