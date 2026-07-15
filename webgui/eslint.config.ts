import { globalIgnores } from 'eslint/config'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'
import pluginOxlint from 'eslint-plugin-oxlint'
import pluginVueI18n from '@intlify/eslint-plugin-vue-i18n'

// To allow more languages other than `ts` in `.vue` files, uncomment the following lines:
// import { configureVueProject } from '@vue/eslint-config-typescript'
// configureVueProject({ scriptLangs: ['ts', 'tsx'] })
// More info at https://github.com/vuejs/eslint-config-typescript/#advanced-setup

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{vue,ts,mts,tsx}'],
  },

  globalIgnores(['**/dist/**', '**/dist-ssr/**', '**/coverage/**']),

  ...pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,

  {
    name: 'app/rules',
    rules: {
      // Page-level view components (Login.vue, Dashboard.vue, ...) are
      // named after their route, not used as custom elements, so the
      // HTML-collision concern this rule guards against doesn't apply.
      'vue/multi-word-component-names': 'off',
    },
  },

  // The plugin's own flat-config types don't line up with
  // defineConfigWithVueTs's stricter ConfigInput type (ecmaVersion typed
  // as number vs the narrower EcmaVersion union) — a type-declaration
  // mismatch only, verified working correctly at runtime.
  ...(pluginVueI18n.configs['flat/recommended'] as unknown as { name: string }[]),
  {
    name: 'app/i18n',
    settings: {
      'vue-i18n': {
        localeDir: './src/locales/*.json',
        messageSyntaxVersion: '^11.0.0',
      },
    },
    rules: {
      // Catches hardcoded template text. Case-by-case exceptions: wrap the
      // element in <!-- eslint-disable-next-line @intlify/vue-i18n/no-raw-text -->
      // with a comment explaining why (e.g. a brand name that's never
      // translated), rather than disabling this broadly.
      '@intlify/vue-i18n/no-raw-text': 'error',
      // A key used in code (t('x.y')) must exist in every locale file.
      '@intlify/vue-i18n/no-missing-keys': 'error',
      // A key present in one locale file must exist in every other one —
      // catches en.json/ru.json drifting out of sync directly, not just
      // via usage.
      '@intlify/vue-i18n/no-missing-keys-in-other-locales': 'error',
      // Keeps locale files from accumulating dead entries as UI changes.
      // settings.fields.* is exempted: SettingField.vue looks those keys
      // up dynamically (t(settingsMeta[key].labelKey)), which the static
      // analyzer can't trace back from a literal string.
      '@intlify/vue-i18n/no-unused-keys': ['error', { ignores: ['/^settings\\.fields\\..*/'] }],
    },
  },

  ...pluginOxlint.buildFromOxlintConfigFile('.oxlintrc.json'),
)
