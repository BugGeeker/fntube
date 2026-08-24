<script setup lang="ts">
import { computed } from 'vue'
import { LockOutlined, UnlockOutlined } from '@ant-design/icons-vue'

const props = defineProps<{
  locked: boolean
}>()
const emit = defineEmits<{
  (e: 'update:locked', val: boolean): void
}>()

const isLocked = computed({
  get: () => props.locked,
  set: (val: boolean) => emit('update:locked', val),
})

function toggle() {
  isLocked.value = !isLocked.value
}
</script>

<template>
  <a-button :type="isLocked ? 'primary' : 'default'" @click="toggle">
    <template #icon>
      <LockOutlined v-if="isLocked" />
      <UnlockOutlined v-else />
    </template>
    <slot />
  </a-button>
</template>
