<script setup lang="ts">
import type { StructuredTableRule, ViewerConfig } from "../types";

defineProps<{ error: string; archiveExtensions: readonly string[] }>();
const emit = defineEmits<{ close: []; save: [] }>();
const settings = defineModel<ViewerConfig>("settings", { required: true });
const rules = defineModel<StructuredTableRule[]>("rules", { required: true });
const categories = ["markdown", "text", "image", "structured"] as const;

function updateExtensions(category: keyof ViewerConfig["extensions"], event: Event) {
  settings.value.extensions[category] = (event.target as HTMLInputElement).value.split(",").map((value) => value.trim()).filter(Boolean);
}

function addRule() {
  rules.value.unshift({ name: "New rule", filePattern: "(?i).*\\.json$", jq: "." });
}

function removeRule(index: number) {
  rules.value.splice(index, 1);
}

function moveRule(index: number, offset: number) {
  const target = index + offset;
  if (target < 0 || target >= rules.value.length) return;
  const [rule] = rules.value.splice(index, 1);
  rules.value.splice(target, 0, rule);
}
</script>

<template>
  <div class="settings-backdrop" @click.self="emit('close')">
    <section class="settings-dialog" role="dialog" aria-modal="true" aria-label="Viewer settings">
      <div class="settings-dialog-header"><div><p class="eyebrow">CONFIGURATION</p><h2>Viewer settings</h2></div><button class="dialog-close" @click="emit('close')">×</button></div>
      <p class="settings-description">表示分類ごとに参照する拡張子をカンマ区切りで設定します。例: <code>.log, .out</code></p>
      <div class="settings-form"><label v-for="category in categories" :key="category" class="settings-extension-field"><span>{{ category }}</span><input :value="settings.extensions[category].join(', ')" @input="updateExtensions(category, $event)" /></label></div>
      <label class="settings-extension-field fixed-extension-field"><span>archive</span><input :value="archiveExtensions.join(', ')" disabled /><small>固定設定</small></label>
      <div class="network-settings">
        <div class="rule-settings-header"><div><p class="eyebrow">NETWORK</p><h3>Proxy and certificate</h3></div></div>
        <p class="rule-settings-help">S3などの接続で使用する設定です。未指定の場合は空のままにします。</p>
        <div class="settings-form">
          <label class="settings-extension-field"><span>Proxy URL</span><input v-model="settings.proxy" placeholder="例: http://proxy.example:8080" spellcheck="false" /></label>
          <label class="settings-extension-field"><span>CA certificate</span><input v-model="settings.certificate" placeholder="証明書ファイルのパス" spellcheck="false" /></label>
        </div>
      </div>
      <div class="rule-settings-header"><div><p class="eyebrow">TABLE RULES</p><h3>Structured table rules</h3></div><button class="text-button rule-add-button" @click="addRule">＋ ルールを追加</button></div>
      <p class="rule-settings-help">上から順に対象ファイルを検索し、最初に一致したjqルールを適用します。</p>
      <p v-if="error" class="settings-error">{{ error }}</p>
      <div class="rule-settings-list">
        <section v-for="(rule, index) in rules" :key="index" class="rule-settings-card">
          <div class="rule-card-header"><strong>{{ index + 1 }}. {{ rule.name || '名称未設定' }}</strong><span><button :disabled="index === 0" title="上へ" @click="moveRule(index, -1)">↑</button><button :disabled="index === rules.length - 1" title="下へ" @click="moveRule(index, 1)">↓</button><button class="rule-remove" title="削除" @click="removeRule(index)">×</button></span></div>
          <label><span>Name</span><input v-model="rule.name" /></label>
          <label><span>File pattern</span><input v-model="rule.filePattern" spellcheck="false" /></label>
          <label><span>jq</span><textarea v-model="rule.jq" rows="3" spellcheck="false"></textarea></label>
        </section>
      </div>
      <div class="settings-actions"><button class="text-button" @click="emit('close')">キャンセル</button><button class="primary-button" @click="emit('save')">設定を保存</button></div>
    </section>
  </div>
</template>
