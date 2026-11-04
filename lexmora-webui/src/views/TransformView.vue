<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import {
  api,
  type ModeOption,
  type OperationOption,
  type TransformOptions,
  type TransformResult,
} from '../api/client'
import Icon from '../components/Icon.vue'
import MarkdownPreview from '../components/MarkdownPreview.vue'
import { formatLocalDateTime } from '../utils/datetime'
import { buildInstructionQuery } from '../utils/instructionKey'
import { startsWithPersian, textDir } from '../utils/textDirection'
import {
  notifyTransformFinished,
  prepareTransformFinishedNotification,
} from '../utils/transformNotification'

const options = ref<TransformOptions | null>(null)
const operation = ref('')
const text = ref('')
const text1 = ref('')
const text2 = ref('')
const direction = ref('')
const mode = ref('')
const topic = ref('')
const movieName = ref('')
const style = ref('')
const language = ref('')

const loading = ref(false)
const optionsLoading = ref(true)
const error = ref('')
const pasteError = ref('')
const result = ref<TransformResult | null>(null)
const copyFeedback = ref('')

const canPaste = typeof navigator !== 'undefined' && !!navigator.clipboard?.readText
const canCopy = typeof navigator !== 'undefined' && !!navigator.clipboard?.writeText

function sortByLabel<T extends { label: string }>(items: T[]): T[] {
  return [...items].sort((a, b) =>
    a.label.localeCompare(b.label, undefined, { sensitivity: 'base' })
  )
}

const sortedOperations = computed(() => sortByLabel(options.value?.operations ?? []))

const currentOp = computed<OperationOption | undefined>(() =>
  options.value?.operations.find((o) => o.value === operation.value)
)

const directions = computed(() => sortByLabel(currentOp.value?.directions ?? []))

const currentDirection = computed(() =>
  directions.value.find((d) => d.value === direction.value)
)

const modes = computed(() => sortByLabel(currentDirection.value?.modes ?? []))

const currentMode = computed<ModeOption | undefined>(() =>
  modes.value.find((m) => m.value === mode.value)
)

const topics = computed(() => sortByLabel(currentMode.value?.topics ?? []))

const styles = computed(() => sortByLabel(currentOp.value?.styles ?? []))

const languages = computed(() => sortByLabel(currentOp.value?.languages ?? []))

const showMovieField = computed(
  () => operation.value === 'translate' && mode.value === 'movie'
)

const showTopicField = computed(() => mode.value === 'scientific' && topics.value.length > 0)

const instructionLink = computed(() => ({
  path: '/instructions',
  query: buildInstructionQuery({
    operation: operation.value,
    direction: direction.value || undefined,
    mode: mode.value || undefined,
    topic: showTopicField.value ? topic.value || undefined : undefined,
    style: style.value || undefined,
    language: language.value || undefined,
  }),
}))

const canSubmit = computed(() => {
  if (operation.value === 'compare') {
    return text1.value.trim().length > 0 && text2.value.trim().length > 0
  }
  return text.value.trim().length > 0 && !!operation.value
})

const inputDir = computed(() => textDir(text.value))
const text1Dir = computed(() => textDir(text1.value))
const text2Dir = computed(() => textDir(text2.value))

const resultDir = computed((): 'rtl' | 'ltr' | 'auto' => {
  if (operation.value === 'translate' || operation.value === 'cursor' || operation.value === 'frontend') {
    if (direction.value === 'english-persian') return 'rtl'
    if (direction.value === 'persian-english' || direction.value === 'english-english') return 'ltr'
  }
  if (operation.value === 'grammar' && language.value === 'fa') return 'rtl'
  if (operation.value === 'grammar' && language.value === 'en') return 'ltr'
  return 'auto'
})

const resultDate = computed(() =>
  result.value
    ? formatLocalDateTime(result.value.created_at, result.value.formatted_date)
    : ''
)

function pickModeDefault(modeList: { value: string }[]): string {
  const general = modeList.find((m) => m.value.toLowerCase() === 'general')
  return general?.value ?? modeList[0]?.value ?? ''
}

function pickTopicDefault(topicList: { value: string }[]): string {
  const general = topicList.find((t) => t.value.toLowerCase() === 'general')
  return general?.value ?? topicList[0]?.value ?? ''
}

function pickOperationDefault(ops: OperationOption[]): string {
  return ops.find((o) => o.value === 'translate')?.value ?? ops[0]?.value ?? ''
}

function pickDefaults(ops: OperationOption[]) {
  if (!ops.length) {
    operation.value = ''
    return
  }
  if (!ops.some((o) => o.value === operation.value)) {
    operation.value = pickOperationDefault(ops)
  }
  syncSecondaryFields()
}

function syncSecondaryFields() {
  const op = currentOp.value
  if (!op) return

  if (op.directions?.length) {
    const preferred =
      op.directions.find((d) => d.value === 'english-persian') ??
      op.directions.find((d) => d.value === 'persian-english') ??
      op.directions[0]
    if (!op.directions.some((d) => d.value === direction.value)) {
      direction.value = preferred.value
    }
    const dir = op.directions.find((d) => d.value === direction.value)
    if (dir?.modes.length) {
      if (!dir.modes.some((m) => m.value === mode.value)) {
        mode.value = pickModeDefault(dir.modes)
      }
    } else {
      mode.value = ''
    }
  } else {
    direction.value = ''
    mode.value = ''
  }

  const modeOpt = currentDirection.value?.modes.find((m) => m.value === mode.value)
  if (modeOpt?.topics?.length) {
    if (!modeOpt.topics.some((t) => t.value === topic.value)) {
      topic.value = pickTopicDefault(modeOpt.topics)
    }
  } else {
    topic.value = ''
  }

  if (op.styles?.length) {
    if (!op.styles.some((s) => s.value === style.value)) {
      style.value = op.styles[0].value
    }
  } else {
    style.value = ''
  }

  if (op.languages?.length) {
    if (!op.languages.some((l) => l.value === language.value)) {
      language.value = op.languages[0].value
    }
  } else if (operation.value === 'term') {
    if (language.value !== 'en' && language.value !== 'fa') {
      language.value = 'en'
    }
  } else {
    language.value = ''
  }
}

function syncDirectionFromText(input: string) {
  if (!input.trim()) return

  if (
    (operation.value === 'translate' || operation.value === 'frontend') &&
    directions.value.length
  ) {
    const target = startsWithPersian(input) ? 'persian-english' : 'english-persian'
    const fallback = startsWithPersian(input) ? 'persian-english' : 'english-english'
    if (directions.value.some((d) => d.value === target) && direction.value !== target) {
      direction.value = target
    } else if (
      directions.value.some((d) => d.value === fallback) &&
      !directions.value.some((d) => d.value === target)
    ) {
      direction.value = fallback
    }
    return
  }

  if (operation.value === 'cursor' && directions.value.length) {
    const pe = directions.value.find((d) => d.value === 'persian-english')
    if (pe) direction.value = pe.value
    return
  }

  if (operation.value === 'term') {
    language.value = startsWithPersian(input) ? 'fa' : 'en'
  }
}

watch(operation, syncSecondaryFields)
watch(direction, syncSecondaryFields)
watch(mode, syncSecondaryFields)
watch(text, (value) => syncDirectionFromText(value))
watch(operation, () => {
  if (text.value.trim()) syncDirectionFromText(text.value)
})

async function loadOptions() {
  optionsLoading.value = true
  error.value = ''
  try {
    options.value = await api.getTransformOptions()
    pickDefaults(options.value.operations)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load options'
  } finally {
    optionsLoading.value = false
  }
}

async function pasteInto(target: 'text' | 'text1' | 'text2') {
  pasteError.value = ''
  if (!canPaste) {
    pasteError.value = 'Clipboard access is not available in this browser.'
    return
  }
  try {
    const clip = await navigator.clipboard.readText()
    if (target === 'text') text.value = clip
    else if (target === 'text1') text1.value = clip
    else text2.value = clip
  } catch {
    pasteError.value = 'Could not read from clipboard. Check browser permissions.'
  }
}

async function copyResult() {
  copyFeedback.value = ''
  if (!result.value?.result_text) return
  if (!canCopy) {
    copyFeedback.value = 'Clipboard unavailable'
    return
  }
  try {
    await navigator.clipboard.writeText(result.value.result_text)
    copyFeedback.value = 'Copied'
    setTimeout(() => {
      if (copyFeedback.value === 'Copied') copyFeedback.value = ''
    }, 1500)
  } catch {
    copyFeedback.value = 'Copy failed'
  }
}

async function submit() {
  if (!canSubmit.value || loading.value) return
  error.value = ''
  result.value = null
  loading.value = true

  const payload: Record<string, unknown> = {
    operation: operation.value,
  }

  if (operation.value === 'compare') {
    payload.text1 = text1.value
    payload.text2 = text2.value
    if (language.value) payload.language = language.value
  } else {
    payload.text = text.value

    if (operation.value === 'translate') {
      payload.direction = direction.value
      payload.mode = mode.value
      if (showTopicField.value) payload.topic = topic.value
      if (showMovieField.value) payload.movie_name = movieName.value
    } else if (operation.value === 'cursor' || operation.value === 'frontend') {
      payload.direction = direction.value
      payload.mode = mode.value
    } else if (operation.value === 'term') {
      payload.style = style.value
      if (language.value) payload.language = language.value
    } else if (operation.value === 'refine') {
      payload.style = style.value
    } else if (operation.value === 'grammar') {
      if (language.value) payload.language = language.value
    }
  }

  await prepareTransformFinishedNotification()

  try {
    result.value = await api.transform(payload)
    notifyTransformFinished(currentOp.value?.label ?? operation.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Transform failed'
  } finally {
    loading.value = false
  }
}

function onSubmitKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    submit()
  }
}

onMounted(loadOptions)
</script>

<template>
  <div class="space-y-6">
    <form class="card space-y-4" @submit.prevent="submit">
      <div v-if="optionsLoading" class="text-sm text-gray-500">Loading options...</div>

      <div v-else class="flex flex-wrap items-end gap-3">
        <div class="w-full max-w-[11rem]">
          <label class="mb-1 block text-sm text-gray-400">Operation</label>
          <select v-model="operation" class="select-field select-compact">
            <option v-for="op in sortedOperations" :key="op.value" :value="op.value">
              {{ op.label }}
            </option>
          </select>
        </div>

        <template v-if="directions.length">
          <div class="w-full max-w-[14rem]">
            <label class="mb-1 block text-sm text-gray-400">Direction</label>
            <select v-model="direction" class="select-field select-compact">
              <option v-for="d in directions" :key="d.value" :value="d.value">
                {{ d.label }}
              </option>
            </select>
          </div>
          <div v-if="modes.length" class="w-full max-w-[11rem]">
            <label class="mb-1 block text-sm text-gray-400">Mode</label>
            <select v-model="mode" class="select-field select-compact">
              <option v-for="m in modes" :key="m.value" :value="m.value">
                {{ m.label }}
              </option>
            </select>
          </div>
          <div v-if="showTopicField" class="w-full max-w-[12rem]">
            <label class="mb-1 block text-sm text-gray-400">Topic</label>
            <select v-model="topic" class="select-field select-compact">
              <option v-for="t in topics" :key="t.value" :value="t.value">
                {{ t.label }}
              </option>
            </select>
          </div>
        </template>

        <template v-if="styles.length">
          <div class="w-full max-w-[11rem]">
            <label class="mb-1 block text-sm text-gray-400">Style</label>
            <select v-model="style" class="select-field select-compact">
              <option v-for="s in styles" :key="s.value" :value="s.value">
                {{ s.label }}
              </option>
            </select>
          </div>
        </template>

        <template v-if="languages.length && operation !== 'term'">
          <div class="w-full max-w-[11rem]">
            <label class="mb-1 block text-sm text-gray-400">Language</label>
            <select v-model="language" class="select-field select-compact">
              <option v-for="l in languages" :key="l.value" :value="l.value">
                {{ l.label === 'en' ? 'English' : l.label === 'fa' ? 'Persian' : l.label }}
              </option>
            </select>
          </div>
        </template>

        <div class="ml-auto">
          <RouterLink :to="instructionLink" class="btn-ghost inline-flex items-center gap-2 text-sm">
            <Icon name="instruction" />
            Instruction
          </RouterLink>
        </div>
      </div>

      <div v-if="showMovieField" class="max-w-md">
        <label class="mb-1 block text-sm text-gray-400">Movie name</label>
        <input v-model="movieName" class="input-field" placeholder="e.g. The Godfather" />
      </div>

      <template v-if="operation === 'compare'">
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <div class="mb-1 flex items-center justify-between gap-2">
              <label class="text-sm text-gray-400">First word / phrase</label>
              <button
                type="button"
                class="btn-ghost px-2 py-1 text-xs"
                :disabled="!canPaste"
                @click="pasteInto('text1')"
              >
                <Icon name="paste" />
                Paste
              </button>
            </div>
            <input
              v-model="text1"
              class="input-field"
              :dir="text1Dir"
              placeholder="e.g. ask"
              @keydown="onSubmitKeydown"
            />
          </div>
          <div>
            <div class="mb-1 flex items-center justify-between gap-2">
              <label class="text-sm text-gray-400">Second word / phrase</label>
              <button
                type="button"
                class="btn-ghost px-2 py-1 text-xs"
                :disabled="!canPaste"
                @click="pasteInto('text2')"
              >
                <Icon name="paste" />
                Paste
              </button>
            </div>
            <input
              v-model="text2"
              class="input-field"
              :dir="text2Dir"
              placeholder="e.g. request"
              @keydown="onSubmitKeydown"
            />
          </div>
        </div>
      </template>

      <div v-else>
        <div class="mb-1 flex items-center justify-between gap-2">
          <label class="text-sm text-gray-400">Input text</label>
          <button
            type="button"
            class="btn-ghost px-2 py-1 text-xs"
            :disabled="!canPaste"
            @click="pasteInto('text')"
          >
            <Icon name="paste" />
            Paste
          </button>
        </div>
        <textarea
          v-model="text"
          rows="6"
          class="input-field resize-y"
          :dir="inputDir"
          placeholder="Enter text to transform... (Enter to submit, Shift+Enter for newline)"
          @keydown="onSubmitKeydown"
        />
      </div>

      <p v-if="pasteError" class="text-sm text-amber-400">{{ pasteError }}</p>
      <p v-if="error" class="text-sm text-red-400">{{ error }}</p>

      <button class="btn-primary" type="submit" :disabled="loading || !canSubmit">
        <Icon name="transform" />
        {{ loading ? 'Processing...' : 'Transform' }}
      </button>
    </form>

    <div v-if="result" class="card space-y-3">
      <div class="flex flex-wrap items-center gap-3 text-sm text-gray-400">
        <span class="rounded bg-accent/20 px-2 py-0.5 text-accent">{{ result.type_display }}</span>
        <span>{{ result.model }}</span>
        <span>{{ resultDate }}</span>
        <button
          type="button"
          class="btn-ghost ml-auto px-2 py-1 text-xs"
          :disabled="!canCopy"
          @click="copyResult"
        >
          <Icon :name="copyFeedback === 'Copied' ? 'check' : 'copy'" />
          {{ copyFeedback || 'Copy' }}
        </button>
      </div>
      <div>
        <h3 class="mb-1 text-sm font-medium text-gray-400">Result</h3>
        <MarkdownPreview :content="result.result_text" :dir="resultDir" />
      </div>
    </div>
  </div>
</template>
