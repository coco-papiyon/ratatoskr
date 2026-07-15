<script setup lang="ts">
import { ref } from "vue";

type ClipboardConversion = {
  input: string;
  output: string;
  inputFormat: string;
};

const props = defineProps<{
  convertToMarkdown: () => Promise<ClipboardConversion>;
  convertToTable: () => Promise<ClipboardConversion>;
}>();
const emit = defineEmits<{ close: [] }>();
const loading = ref(false);
const error = ref("");
const result = ref<ClipboardConversion>();
const direction = ref<"markdown" | "table">("markdown");

async function convertClipboard(target: "markdown" | "table") {
  loading.value = true;
  error.value = "";
  try {
    direction.value = target;
    result.value = target === "markdown" ? await props.convertToMarkdown() : await props.convertToTable();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="settings-backdrop" @click.self="emit('close')">
    <section class="clipboard-dialog" role="dialog" aria-modal="true" aria-label="クリップボードの表を変換">
      <div class="settings-dialog-header">
        <div><p class="eyebrow">CLIPBOARD TABLE</p><h2>クリップボード表変換</h2></div>
        <button class="dialog-close" aria-label="閉じる" @click="emit('close')">×</button>
      </div>
      <p class="settings-description">Excelなどの表をMarkdownへ変換、またはMarkdown表をExcelへ貼り付けられるタブ区切りへ変換します。</p>
      <p v-if="error" class="settings-error">{{ error }}</p>
      <div class="clipboard-dialog-actions clipboard-direction-actions">
        <span v-if="result" class="clipboard-format">入力形式: {{ result.inputFormat }}</span>
        <button class="primary-button direction-button" :disabled="loading" @click="convertClipboard('markdown')">{{ loading && direction === 'markdown' ? '変換中...' : '表 → Markdown' }}</button>
        <button class="primary-button" :disabled="loading" @click="convertClipboard('table')">{{ loading && direction === 'table' ? '変換中...' : 'Markdown → 表' }}</button>
      </div>
      <div class="clipboard-preview-grid">
        <article class="clipboard-preview-card"><header>変換前</header><pre>{{ result?.input || 'まだ変換していません。' }}</pre></article>
        <article class="clipboard-preview-card"><header>{{ direction === 'markdown' ? '変換後 Markdown' : '変換後 タブ区切り' }}</header><pre>{{ result?.output || '変換結果がここに表示されます。' }}</pre></article>
      </div>
      <div class="settings-actions"><button class="text-button" @click="emit('close')">閉じる</button></div>
    </section>
  </div>
</template>
