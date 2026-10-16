<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { api, type CreditsInfo, type OpenRouterModel } from '../api/client'
import Icon from '../components/Icon.vue'

type ApiProvider = 'openrouter' | 'gemini' | 'opencode_go'

const PROVIDER_STORAGE_KEY = 'lexmora_api_provider'
const DEFAULT_GEMINI_MODEL = 'gemini-2.0-flash'

const apiProvider = ref<ApiProvider>('openrouter')
const apiKey = ref('')
const geminiApiKey = ref('')
const openCodeGoApiKey = ref('')
const modelName = ref('')
const modelQuery = ref('')
const modelResults = ref<OpenRouterModel[]>([])
const showModelResults = ref(false)
const modelsLoading = ref(false)
const credits = ref<CreditsInfo | null>(null)
const creditsError = ref('')
const loading = ref(true)
const saving = ref(false)
const clearing = ref(false)
const error = ref('')
const saved = ref(false)

const isOpenRouter = computed(() => apiProvider.value === 'openrouter')
const isGemini = computed(() => apiProvider.value === 'gemini')
const isOpenCodeGo = computed(() => apiProvider.value === 'opencode_go')
const hasModelSearch = computed(() => isOpenRouter.value || isOpenCodeGo.value)

let searchTimer: ReturnType<typeof setTimeout> | null = null

function readStoredProvider(): ApiProvider {
  const stored = localStorage.getItem(PROVIDER_STORAGE_KEY)
  return stored === 'gemini' || stored === 'opencode_go' ? stored : 'openrouter'
}

function persistProvider(provider: ApiProvider) {
  localStorage.setItem(PROVIDER_STORAGE_KEY, provider)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const settings = await api.getSettings()
    apiProvider.value =
      settings.api_provider === 'gemini' ||
      settings.api_provider === 'openrouter' ||
      settings.api_provider === 'opencode_go'
        ? settings.api_provider
        : readStoredProvider()
    persistProvider(apiProvider.value)
    apiKey.value = settings.openrouter_api_key
    geminiApiKey.value = settings.gemini_api_key ?? ''
    openCodeGoApiKey.value = settings.opencode_go_api_key ?? ''
    modelName.value = settings.model_name
    modelQuery.value = settings.model_name
    if (isOpenRouter.value) {
      await Promise.all([loadCredits(), searchModels('')])
    } else if (isOpenCodeGo.value) {
      await searchModels('')
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load settings'
  } finally {
    loading.value = false
  }
}

async function loadCredits() {
  creditsError.value = ''
  try {
    credits.value = await api.getCredits()
  } catch (e) {
    credits.value = null
    creditsError.value = e instanceof Error ? e.message : 'Failed to load credits'
  }
}

async function searchModels(q: string) {
  modelsLoading.value = true
  try {
    modelResults.value = await api.searchModels(q)
  } catch {
    modelResults.value = []
  } finally {
    modelsLoading.value = false
  }
}

function scheduleModelSearch(q: string) {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => searchModels(q), 300)
}

watch(modelQuery, (q) => {
  if (!hasModelSearch.value) return
  if (q !== modelName.value) {
    scheduleModelSearch(q)
    showModelResults.value = true
  }
})

watch(apiProvider, (provider, prev) => {
  persistProvider(provider)
  if (provider === 'gemini' && prev === 'openrouter' && !loading.value) {
    if (!modelQuery.value.trim() || modelQuery.value.includes('/')) {
      modelQuery.value = DEFAULT_GEMINI_MODEL
    }
  }
  if (provider === 'openrouter' && !loading.value) {
    loadCredits()
    searchModels(modelQuery.value)
  }
  if (provider === 'opencode_go' && !loading.value) {
    modelResults.value = []
  }
})

function selectModel(model: OpenRouterModel) {
  modelName.value = model.id
  modelQuery.value = model.id
  showModelResults.value = false
}

function formatCredits(n: number | null | undefined) {
  if (n == null || Number.isNaN(n)) return '—'
  return n.toFixed(4).replace(/\.?0+$/, '')
}

async function save() {
  saving.value = true
  error.value = ''
  saved.value = false
  try {
    const selectedModel = modelQuery.value.trim() || modelName.value.trim()
    persistProvider(apiProvider.value)
    await api.updateSettings({
      openrouter_api_key: apiKey.value,
      gemini_api_key: geminiApiKey.value,
      opencode_go_api_key: openCodeGoApiKey.value,
      model_name: selectedModel,
      api_provider: apiProvider.value,
    })
    modelName.value = selectedModel
    modelQuery.value = selectedModel
    saved.value = true
    if (isOpenRouter.value) await loadCredits()
    if (isOpenCodeGo.value) await searchModels('')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Save failed'
  } finally {
    saving.value = false
  }
}

async function clearAll() {
  if (!confirm('Delete ALL history records? This cannot be undone.')) return
  clearing.value = true
  error.value = ''
  try {
    await api.clearData()
    alert('All history records have been deleted.')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Clear failed'
  } finally {
    clearing.value = false
  }
}

function onModelBlur() {
  setTimeout(() => {
    showModelResults.value = false
  }, 150)
}

onMounted(load)
onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<template>
  <div class="space-y-6">
    <div v-if="loading" class="text-gray-500">Loading...</div>

    <div v-else class="card max-w-xl space-y-4">
      <div>
        <label class="mb-1 block text-sm text-gray-400">API provider</label>
        <select v-model="apiProvider" class="select-field select-compact">
          <option value="openrouter">OpenRouter</option>
          <option value="gemini">Google Gemini</option>
          <option value="opencode_go">OpenCode Go</option>
        </select>
      </div>

      <template v-if="isOpenRouter">
        <div>
          <label class="mb-1 block text-sm text-gray-400">OpenRouter API token</label>
          <input v-model="apiKey" type="password" class="input-field" placeholder="sk-or-..." />
        </div>

        <div class="relative">
          <label class="mb-1 block text-sm text-gray-400">Model</label>
          <input
            v-model="modelQuery"
            class="input-field"
            placeholder="Search OpenRouter models..."
            autocomplete="off"
            @focus="showModelResults = true"
            @blur="onModelBlur"
          />
          <p class="mt-1 text-xs text-gray-500">
            Selected: <span class="text-gray-300">{{ modelName || 'none' }}</span>
          </p>
          <div
            v-if="showModelResults && (modelsLoading || modelResults.length)"
            class="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-surface-border bg-surface shadow-lg"
          >
            <div v-if="modelsLoading" class="px-3 py-2 text-sm text-gray-500">Searching...</div>
            <button
              v-for="m in modelResults"
              :key="m.id"
              type="button"
              class="block w-full px-3 py-2 text-left text-sm hover:bg-accent/20"
              @mousedown.prevent="selectModel(m)"
            >
              <span class="text-white">{{ m.name }}</span>
              <span class="mt-0.5 block text-xs text-gray-500">{{ m.id }}</span>
            </button>
          </div>
        </div>

        <div class="rounded-lg border border-surface-border bg-surface/50 p-3">
          <h3 class="mb-2 text-sm font-medium text-gray-300">OpenRouter credits</h3>
          <p v-if="creditsError" class="text-sm text-red-400">{{ creditsError }}</p>
          <template v-else-if="credits">
            <p v-if="credits.remaining != null" class="text-sm text-white">
              Remaining: <span class="text-accent">{{ formatCredits(credits.remaining) }}</span>
            </p>
            <p v-if="credits.source === 'credits'" class="mt-1 text-xs text-gray-500">
              Purchased {{ formatCredits(credits.total_credits) }} · Used
              {{ formatCredits(credits.total_usage) }}
            </p>
            <p v-else class="mt-1 text-xs text-gray-500">
              Key usage {{ formatCredits(credits.usage) }}
              <span v-if="credits.limit_remaining != null">
                · Key limit remaining {{ formatCredits(credits.limit_remaining) }}
              </span>
              <span v-else> · Account balance needs an OpenRouter management key</span>
            </p>
          </template>
          <p v-else class="text-sm text-gray-500">Save an API token to see credits.</p>
        </div>
      </template>

      <template v-else-if="isGemini">
        <div>
          <label class="mb-1 block text-sm text-gray-400">Gemini API key</label>
          <input
            v-model="geminiApiKey"
            type="password"
            class="input-field"
            placeholder="AIza..."
          />
        </div>

        <div>
          <label class="mb-1 block text-sm text-gray-400">Model</label>
          <input
            v-model="modelQuery"
            class="input-field"
            :placeholder="DEFAULT_GEMINI_MODEL"
            autocomplete="off"
          />
          <p class="mt-1 text-xs text-gray-500">
            Selected: <span class="text-gray-300">{{ modelQuery || modelName || 'none' }}</span>
          </p>
        </div>
      </template>

      <template v-else-if="isOpenCodeGo">
        <div>
          <label class="mb-1 block text-sm text-gray-400">OpenCode Go API key</label>
          <input
            v-model="openCodeGoApiKey"
            type="password"
            class="input-field"
            placeholder="oc_sk_..."
          />
        </div>

        <div class="relative">
          <label class="mb-1 block text-sm text-gray-400">Model</label>
          <input
            v-model="modelQuery"
            class="input-field"
            placeholder="Search OpenCode Go models..."
            autocomplete="off"
            @focus="showModelResults = true"
            @blur="onModelBlur"
          />
          <p class="mt-1 text-xs text-gray-500">
            Selected: <span class="text-gray-300">{{ modelName || 'none' }}</span>
          </p>
          <p class="mt-1 text-xs text-gray-500">Save the API key to load the model list.</p>
          <div
            v-if="showModelResults && (modelsLoading || modelResults.length)"
            class="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-surface-border bg-surface shadow-lg"
          >
            <div v-if="modelsLoading" class="px-3 py-2 text-sm text-gray-500">Searching...</div>
            <button
              v-for="m in modelResults"
              :key="m.id"
              type="button"
              class="block w-full px-3 py-2 text-left text-sm hover:bg-accent/20"
              @mousedown.prevent="selectModel(m)"
            >
              <span class="text-white">{{ m.name }}</span>
              <span class="mt-0.5 block text-xs text-gray-500">{{ m.id }}</span>
            </button>
          </div>
        </div>
      </template>

      <div class="flex items-center gap-4">        <button class="btn-primary" :disabled="saving" @click="save">
          <Icon name="save" />
          {{ saving ? 'Saving...' : 'Save settings' }}
        </button>
        <span v-if="saved" class="text-sm text-green-400">Saved</span>
      </div>

      <p v-if="error" class="text-sm text-red-400">{{ error }}</p>
    </div>

    <div class="card max-w-xl border-red-900/50">
      <h3 class="mb-2 font-medium text-red-400">Danger zone</h3>
      <p class="mb-4 text-sm text-gray-400">
        Remove all rows from the history table. Instructions and settings are kept.
      </p>
      <button class="btn-danger" :disabled="clearing" @click="clearAll">
        <Icon name="clear" />
        {{ clearing ? 'Clearing...' : 'Clear all history' }}
      </button>
    </div>
  </div>
</template>
