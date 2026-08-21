<template>
  <a-card title="飞牛影视配置">
    <a-form :model="form" layout="vertical" @finish="handleSave">
      <a-form-item label="服务端地址" name="host" required>
        <a-input
          v-model:value="form.host"
          placeholder="如 http://127.0.0.1:5666/v"
          allow-clear
        />
      </a-form-item>
      <a-form-item label="用户名" name="username" required>
        <a-input v-model:value="form.username" placeholder="飞牛影视用户名" allow-clear />
      </a-form-item>
      <a-form-item label="密码" name="password" required>
        <a-input-password v-model:value="form.password" placeholder="飞牛影视密码" allow-clear />
      </a-form-item>
      <a-form-item label="访问码" name="access_code">
        <a-input v-model:value="form.access_code" placeholder="未开启访问码则留空" allow-clear />
      </a-form-item>
      <a-form-item label="外网播放地址" name="play_host">
        <a-input v-model:value="form.play_host" placeholder="如 http://your-domain:5666/v" allow-clear />
      </a-form-item>
      <a-form-item label="同步媒体库" name="sync_libraries">
        <a-input
          v-model:value="form.sync_libraries"
          placeholder='JSON数组，如 ["guid1","guid2"]，或 ["all"]'
          allow-clear
        />
      </a-form-item>
      <a-space>
        <a-button type="primary" html-type="submit" :loading="saving">保存配置</a-button>
        <a-button :loading="testing" @click="handleTest">测试连接</a-button>
      </a-space>
    </a-form>

    <a-alert
      v-if:="testResult"
      style="margin-top: 16px"
      :type="testResult.success ? 'success' : 'error'"
      :message="testResult.message"
      show-icon
    />
  </a-card>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { message } from 'ant-design-vue'
import { useTrimMediaStore } from '@/stores/trimmedia'
import type { TrimMediaConfig } from '@/api/trimmedia'

const store = useTrimMediaStore()

const form = reactive<TrimMediaConfig>({
  host: '',
  username: '',
  password: '',
  access_code: '',
  play_host: '',
  sync_libraries: '',
})

const saving = ref(false)
const testing = ref(false)
const testResult = ref<{ success: boolean; message: string } | null>(null)

onMounted(async () => {
  try {
    await store.fetchConfig()
    if (store.config) {
      Object.assign(form, store.config)
    }
  } catch {
    // 配置尚未保存
  }
})

async function handleSave() {
  if (!form.host || !form.username || !form.password) {
    message.warning('请填写服务端地址、用户名和密码')
    return
  }
  saving.value = true
  try {
    await store.saveConfigData({ ...form })
    message.success('配置已保存，服务将在后台重连')
  } catch (e: any) {
    message.error('保存失败: ' + (e?.message || ''))
  } finally {
    saving.value = false
  }
}

async function handleTest() {
  if (!form.host || !form.username || !form.password) {
    message.warning('请填写服务端地址、用户名和密码')
    return
  }
  testing.value = true
  testResult.value = null
  try {
    const result = await store.testConnectionData({ ...form })
    testResult.value = {
      success: true,
      message: `连接成功！前端版本: ${result.version?.frontend || '未知'}, 后端版本: ${result.version?.backend || '未知'}`,
    }
  } catch (e: any) {
    testResult.value = { success: false, message: '连接失败: ' + (e?.message || '请检查配置') }
  } finally {
    testing.value = false
  }
}
</script>
