<script setup lang="ts">
import { ref } from 'vue'
import axios from 'axios'

const loading = ref(false)
const response = ref<unknown>(null)
const error = ref<string | null>(null)

const callMedlemsservice = async () => {
  loading.value = true
  error.value = null
  response.value = null
  try {
    // The `session_id` cookie is set by medlem.dds.dk when the user is logged in
    // there. By setting `withCredentials: true` the browser will automatically
    // attach that cookie to the cross-origin request, so there is no need to
    // read or forward it manually (and cross-origin cookies aren't readable
    // from JS anyway).
    const res = await axios.post(
      'https://medlem.dds.dk/web/session/get_session_info',
      {
        jsonrpc: '2.0',
        method: 'call',
        id: 1,
        params: {},
      },
      {
        withCredentials: true,
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )
    response.value = res.data
  } catch (err: unknown) {
    if (axios.isAxiosError(err)) {
      error.value = err.response
        ? `${err.response.status} ${err.response.statusText}: ${JSON.stringify(err.response.data)}`
        : err.message
    } else if (err instanceof Error) {
      error.value = err.message
    } else {
      error.value = 'Unknown error'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="py-4">
    <h1 class="font-nathejk text-2xl pb-4">Login</h1>

    <Button
      label="Medlemsservice"
      icon="pi pi-sign-in"
      :loading="loading"
      @click="callMedlemsservice"
    />

    <div v-if="error" class="mt-4 p-3 border border-red-400 bg-red-50 text-red-800 rounded">
      <strong>Fejl:</strong>
      <pre class="whitespace-pre-wrap break-all text-sm">{{ error }}</pre>
    </div>

    <div v-if="response !== null" class="mt-4">
      <h2 class="font-semibold pb-2">Svar</h2>
      <pre
        class="p-3 bg-gray-100 text-gray-900 rounded text-sm overflow-auto whitespace-pre-wrap break-all"
      >{{ JSON.stringify(response, null, 2) }}</pre>
    </div>
  </div>
</template>
