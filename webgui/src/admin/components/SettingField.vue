<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Password from 'primevue/password'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import Tag from 'primevue/tag'
import type { Setting } from '@/types/settings'

const props = defineProps<{
  setting: Setting
  labelKey: string
  hintKey: string
  modelValue: string
}>()

const emit = defineEmits<{ 'update:modelValue': [string] }>()

const { t } = useI18n()

// env-only (never editable) or currently shadowed by its env var (would
// become editable if that env var were removed and the process
// restarted) — both render as read-only, but for different reasons, so
// they get distinct tags/hints.
const readOnly = computed(() => !props.setting.editable || props.setting.env_locked)

const numberValue = computed<number | null>({
  get: () => (props.modelValue === '' ? null : Number(props.modelValue)),
  set: (v) => emit('update:modelValue', v === null ? '' : String(v)),
})

const boolValue = computed({
  get: () => props.modelValue === 'true',
  set: (v: boolean) => emit('update:modelValue', v ? 'true' : 'false'),
})
</script>

<template>
  <div class="flex flex-col gap-1">
    <div class="flex flex-wrap items-center gap-1.5">
      <label class="text-sm font-medium">{{ t(labelKey) }}</label>
      <Tag
        v-if="!setting.editable"
        severity="secondary"
        :value="t('settings.env_only')"
        :title="t('settings.env_only_hint', { var: setting.env_var })"
      />
      <Tag
        v-else-if="setting.env_locked"
        severity="warn"
        :value="t('settings.env_locked')"
        :title="t('settings.env_locked_hint', { var: setting.env_var })"
      />
    </div>
    <p class="m-0 text-xs text-muted-color">{{ t(hintKey) }}</p>

    <p v-if="readOnly && setting.secret" class="m-0 text-sm">{{ t('settings.secret_configured') }}</p>
    <p v-else-if="readOnly" class="m-0 text-sm">{{ setting.value || setting.default }}</p>

    <Password
      v-else-if="setting.type === 'password'"
      :model-value="modelValue"
      :feedback="false"
      toggle-mask
      fluid
      :placeholder="t('settings.secret_hint')"
      @update:model-value="emit('update:modelValue', $event ?? '')"
    />
    <InputNumber
      v-else-if="setting.type === 'int'"
      v-model="numberValue"
      fluid
    />
    <ToggleSwitch
      v-else-if="setting.type === 'bool'"
      v-model="boolValue"
    />
    <Select
      v-else-if="setting.type === 'select'"
      :model-value="modelValue"
      :options="setting.options"
      fluid
      @update:model-value="emit('update:modelValue', $event)"
    />
    <InputText
      v-else
      :model-value="modelValue"
      fluid
      @update:model-value="emit('update:modelValue', $event ?? '')"
    />
  </div>
</template>
