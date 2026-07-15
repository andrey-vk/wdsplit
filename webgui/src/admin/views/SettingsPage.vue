<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave } from 'vue-router'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useToast } from 'primevue/usetoast'
import apiClient from '@/api/client'
import { settingsListSchema, type Setting } from '@/types/settings'
import { settingsMeta } from '@/admin/settingsMeta'
import SettingField from '@/admin/components/SettingField.vue'

const { t } = useI18n()
const toast = useToast()

const settingsList = ref<Setting[]>([])
const edits = ref<Record<string, string>>({})
const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const saved = ref(false)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const resp = await apiClient.get('/admin/settings')
    const parsed = settingsListSchema.parse(resp.data)
    settingsList.value = parsed
    edits.value = Object.fromEntries(parsed.map((s) => [s.key, s.value]))
  } catch {
    loadError.value = t('settings.load_error')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function isChanged(s: Setting): boolean {
  return s.editable && !s.env_locked && edits.value[s.key] !== s.value
}

const dirty = computed(() => settingsList.value.some(isChanged))

onBeforeRouteLeave(() => {
  if (dirty.value) {
    return window.confirm(t('settings.unsaved_changes_confirm'))
  }
  return true
})

async function save() {
  const changed: Record<string, string> = {}
  for (const s of settingsList.value) {
    if (isChanged(s)) changed[s.key] = edits.value[s.key] ?? ''
  }
  if (Object.keys(changed).length === 0) return

  saving.value = true
  saved.value = false
  try {
    const resp = await apiClient.put('/admin/settings', changed)
    if (resp.data.ok) {
      saved.value = true
      await load()
    } else {
      toast.add({ severity: 'error', summary: t('settings.save_error'), detail: resp.data.error, life: 5000 })
    }
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    toast.add({ severity: 'error', summary: t('settings.save_error'), detail: e.response?.data?.error, life: 5000 })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="max-w-2xl p-8">
    <h1 class="mb-6 text-2xl font-bold">{{ t('settings.title') }}</h1>

    <div v-if="loading" class="flex justify-center py-12">
      <i class="pi pi-spinner pi-spin text-2xl" />
    </div>

    <Message v-else-if="loadError" severity="error" :closable="false">{{ loadError }}</Message>

    <template v-else>
      <Message v-if="saved" severity="success" :closable="false" class="mb-4">{{ t('settings.saved') }}</Message>

      <div
        class="flex flex-col gap-6 rounded-xl border border-surface-200 bg-surface-0 p-6 dark:border-surface-800 dark:bg-surface-900"
      >
        <SettingField
          v-for="setting in settingsList"
          :key="setting.key"
          :model-value="edits[setting.key] ?? ''"
          :setting="setting"
          :label-key="settingsMeta[setting.key]?.labelKey ?? setting.key"
          :hint-key="settingsMeta[setting.key]?.hintKey ?? ''"
          @update:model-value="edits[setting.key] = $event"
        />
      </div>

      <div class="sticky bottom-4 mt-6 flex justify-end">
        <Button
          :label="t('settings.save')"
          icon="pi pi-check"
          :loading="saving"
          :disabled="!dirty"
          @click="save"
        />
      </div>
    </template>
  </div>
</template>
