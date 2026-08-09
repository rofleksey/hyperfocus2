<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import Column from "primevue/column";
import DataTable from "primevue/datatable";
import Tag from "primevue/tag";
import { useRoute } from "vue-router";
import { fetchStreamer, type StreamerSummary, type StreamerSession } from "../api";

const props = defineProps<{ id: string }>();
const route = useRoute();

const streamer = ref<StreamerSummary | null>(null);
const sessions = ref<StreamerSession[]>([]);
const loading = ref(false);
const error = ref("");

function fmt(s?: string): string {
  if (!s) return "—";
  try {
    return new Date(s).toLocaleString();
  } catch {
    return s;
  }
}

function duration(start: string, end?: string): string {
  const a = new Date(start).getTime();
  const b = end ? new Date(end).getTime() : Date.now();
  const mins = Math.max(0, Math.round((b - a) / 60000));
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

async function load(id: string) {
  loading.value = true;
  error.value = "";
  try {
    const res = await fetchStreamer(id);
    streamer.value = res.streamer;
    sessions.value = res.sessions ?? [];
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

watch(() => props.id ?? (route.params.id as string), (v) => v && load(v));
onMounted(() => load(props.id ?? (route.params.id as string)));
</script>

<template>
  <section>
    <h2 v-if="streamer">{{ streamer.display_name }}</h2>
    <h2 v-else>Streamer</h2>
    <p v-if="streamer" class="muted">@{{ streamer.login }}</p>

    <p v-if="error" class="muted">Error: {{ error }}</p>

    <h3 style="margin-top: 1.5rem">Recent sessions</h3>
    <DataTable :value="sessions" :loading="loading" stripedRows paginator :rows="25" size="small">
      <Column field="started_at" header="Started">
        <template #body="{ data }">{{ fmt(data.started_at) }}</template>
      </Column>
      <Column field="ended_at" header="Ended">
        <template #body="{ data }">{{ fmt(data.ended_at) }}</template>
      </Column>
      <Column header="Duration">
        <template #body="{ data }">{{ duration(data.started_at, data.ended_at) }}</template>
      </Column>
      <Column header="Status" style="width: 120px">
        <template #body="{ data }">
          <Tag v-if="!data.ended_at" value="LIVE" severity="success" />
          <Tag v-else value="offline" severity="secondary" />
        </template>
      </Column>
    </DataTable>
  </section>
</template>
