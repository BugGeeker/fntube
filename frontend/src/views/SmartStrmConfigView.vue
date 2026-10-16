<template>
  <a-form :model="form" layout="vertical" @finish="handleSave">
    <a-card title="SmartStrm设置">
      <a-form-item label="服务地址" name="url" required>
        <a-input v-model:value="form.url" placeholder="如 http://127.0.0.1:8080" allow-clear />
      </a-form-item>
      <a-form-item label="账号" name="username" required>
        <a-input v-model:value="form.username" placeholder="SmartStrm 登录账号" allow-clear />
      </a-form-item>
      <a-form-item label="密码" name="password" required>
        <a-input-password v-model:value="form.password" placeholder="SmartStrm 登录密码" allow-clear />
      </a-form-item>
      <a-form-item label="strm 根目录" name="strm_root_path" required>
        <a-input
          v-model:value="form.strm_root_path"
          placeholder="如 /mnt/media/strm"
          allow-clear
        />
      </a-form-item>
    </a-card>
    <a-space style="margin-top: 16px">
      <a-button type="primary" html-type="submit" :loading="saving">
        保存配置
      </a-button>
      <a-button :loading="testing" @click="handleTest">
        测试连接
      </a-button>
    </a-space>
    <a-alert
      v-if="testResult"
      style="margin-top: 16px"
      :type="testResult.success ? 'success' : 'error'"
      :message="testResult.message"
      show-icon
    />
  </a-form>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  getSmartStrmConfig,
  saveSmartStrmConfig,
  testSmartStrmConnection,
  type SmartStrmConfig,
} from '@/api/smartstrm'

const form = reactive<SmartStrmConfig>({
  url: '',
  username: '',
  password: '',
  strm_root_path: '',
})
const saving = ref(false)
const testing = ref(false)
const testResult = ref<{ success: boolean; message: string } | null>(null)

onMounted(async () => {
  try {
    const { data } = await getSmartStrmConfig()
    Object.assign(form, data)
  } catch (error: any) {
    message.error(error?.response?.data?.error || '读取 SmartStrm 配置失败')
  }
})

async function handleSave() {
  if (!form.url.trim() || !form.username.trim() || !form.password || !form.strm_root_path.trim()) {
    message.warning('请填写完整的 SmartStrm 配置')
    return
  }
  saving.value = true
  try {
    const { data } = await saveSmartStrmConfig({
      ...form,
      url: form.url.trim(),
      username: form.username.trim(),
      strm_root_path: form.strm_root_path.trim(),
    })
    Object.assign(form, data.config)
    message.success('SmartStrm 配置已保存')
  } catch (error: any) {
    message.error(error?.response?.data?.error || error?.message || '保存 SmartStrm 配置失败')
  } finally {
    saving.value = false
  }
}

async function handleTest() {
  if (!form.url.trim() || !form.username.trim() || !form.password) {
    message.warning('请填写 SmartStrm 地址、账号和密码')
    return
  }

  testing.value = true
  testResult.value = null
  try {
    const { data } = await testSmartStrmConnection({
      url: form.url.trim(),
      username: form.username.trim(),
      password: form.password,
    })
    const versionText = data.remote_version && data.remote_version !== data.version
      ? `当前版本: ${data.version}，最新版本: ${data.remote_version}`
      : `当前版本: ${data.version}`
    testResult.value = {
      success: true,
      message: `连接成功！${versionText}${data.is_new ? '，有新版本可用' : ''}`,
    }
  } catch (error: any) {
    const detail = error?.response?.data?.error
    testResult.value = {
      success: false,
      message: detail || `连接失败: ${error?.message || '请检查配置'}`,
    }
  } finally {
    testing.value = false
  }
}
</script>
