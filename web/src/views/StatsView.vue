<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Chart, LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler } from "chart.js";
import { Line } from "vue-chartjs";
import { fetchStats, type SnapshotStat } from "../api";

Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler);

const stats = ref<SnapshotStat[]>([]);
const loading = ref(false);
const error = ref("");

function fmtLabel(raw: string): string {
  try {
    return new Date(raw).toLocaleTimeString();
  } catch {
    return raw;
  }
}

const onlineData = ref(makeEmpty());
const durationData = ref(makeEmpty());
const diskData = ref(makeEmpty());
const previewData = ref(makeEmpty());
const ocrData = ref(makeEmpty());

function makeEmpty() {
  return { labels: [], datasets: [] as { label: string; data: number[]; borderColor: string; backgroundColor: string; fill: boolean; tension: number; pointRadius: number }[] };
}

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { display: false } },
  scales: {
    x: { ticks: { color: "#94a3b8", font: { size: 10 } }, grid: { color: "#1e293b" } },
    y: { ticks: { color: "#94a3b8", font: { size: 10 } }, grid: { color: "#1e293b" }, beginAtZero: false },
  },
};

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const res = await fetchStats(100);
    stats.value = res.snapshots;
    if (!stats.value.length) return;

    const labs = stats.value.map(s => fmtLabel(s.taken_at));

    onlineData.value = dataset(labs, "Streams online", stats.value.map(s => s.stream_count), "#6366f1", "rgba(99,102,241,0.1)");
    durationData.value = dataset(labs, "Cycle time (s)", stats.value.map(s => s.duration_seconds), "#ec4899", "rgba(236,72,153,0.1)");
    diskData.value = dataset(labs, "Disk (MB)", stats.value.map(s => +(s.disk_usage_bytes / 1048576).toFixed(1)), "#8b5cf6", "rgba(139,92,246,0.1)");
    previewData.value = dataset(labs, "Previews (%)", stats.value.map(s => s.total > 0 ? Math.round((s.preview_ok / s.total) * 100) : 0), "#22c55e", "rgba(34,197,94,0.1)");
    ocrData.value = dataset(labs, "OCR names (%)", stats.value.map(s => s.total > 0 ? Math.round((s.ocr_ok / s.total) * 100) : 0), "#f59e0b", "rgba(245,158,11,0.1)");
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function dataset(labels: string[], label: string, data: number[], border: string, bg: string) {
  return {
    labels,
    datasets: [{ label, data, borderColor: border, backgroundColor: bg, fill: true, tension: 0.3, pointRadius: 0 }],
  };
}

onMounted(load);
</script>

<template>
  <section class="stats-page">
    <h2>Stats</h2>
    <p v-if="error" class="muted">Error: {{ error }}</p>
    <div v-if="loading" class="loading-spinner"><span class="spinner"></span></div>

    <template v-if="!loading && stats.length">
      <div class="chart-box">
        <h3>Streams online</h3>
        <div class="chart-wrap"><Line v-if="onlineData.labels.length" :data="onlineData" :options="chartOptions" /></div>
      </div>

      <div class="chart-box">
        <h3>Cycle time (seconds)</h3>
        <div class="chart-wrap"><Line v-if="durationData.labels.length" :data="durationData" :options="chartOptions" /></div>
      </div>

      <div class="chart-box">
        <h3>Disk usage (MB)</h3>
        <div class="chart-wrap"><Line v-if="diskData.labels.length" :data="diskData" :options="chartOptions" /></div>
      </div>

      <div class="chart-box">
        <h3>Preview capture rate</h3>
        <div class="chart-wrap"><Line v-if="previewData.labels.length" :data="previewData" :options="chartOptions" /></div>
      </div>

      <div class="chart-box">
        <h3>OCR success rate</h3>
        <div class="chart-wrap"><Line v-if="ocrData.labels.length" :data="ocrData" :options="chartOptions" /></div>
      </div>
    </template>
  </section>
</template>

<style scoped>
.stats-page { max-width: 100%; margin: 0 auto; }
.stats-page h2 { margin-top: 0; font-size: 1.1rem; }
.stats-page h3 { font-size: 0.9rem; margin: 1.25rem 0 0.5rem; color: var(--p-text-muted-color); }

.chart-box { margin-bottom: 0.5rem; }
.chart-wrap { height: 220px; }

.loading-spinner { display: flex; justify-content: center; padding: 2rem 0; }
.spinner {
  width: 32px; height: 32px;
  border: 3px solid var(--p-surface-700, #374151);
  border-top-color: var(--p-primary-color, #6366f1);
  border-radius: 50%; animation: spin 0.7s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
