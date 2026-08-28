<template>
  <div class="img-wrapper">
    <img v-if="src && !uiStore.hideImages" :src="proxyImage(src)" class="media-image" :alt="alt" />
    <div v-else class="media-image media-image-placeholder">
      <span class="media-image-placeholder-text">{{ fallbackText }}</span>
    </div>
  </div>

</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'
import { proxyImage } from '@/utils/image'
import { theme } from 'ant-design-vue'

const props = withDefaults(defineProps<{
  src?: string | null
  alt?: string
  ratio?: string
  fallback?: string
  fit?: string
  placeholderFontSize?: string
  borderRadius?: string
}>(), {
  alt: 'poster',
  ratio: '2 / 3',
  fallback: '',
  fit: 'cover',
  placeholderFontSize: '24px',
  borderRadius: '8px',
})

const uiStore = useUiStore()

const { token } = theme.useToken()

const fallbackText = computed(() => {
  if (props.fallback) return props.fallback
  return (props.alt || '?').split(' ')[0]
})
</script>

<style scoped>
.img-wrapper {
  background: v-bind('token.colorBgLayout');
}

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
  background: v-bind('token.colorBorderSecondary');
}

.media-image-placeholder-text {
  padding: 0 12px;
  color: v-bind('token.colorTextTertiary');
  font-size: v-bind('props.placeholderFontSize');
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
