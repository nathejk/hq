<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { http } from '@/plugins/axios';

const toast = useToast();

onMounted(() => load())

const config = ref({patruljeCount:0})
const load = async () => {
  try {
    const response = await http.get('/home');
    config.value = response.data.config;
    console.log('reso', response.data.config);
  } catch (error) {
    console.log('home load failed', error);
  }
}
</script>

<template>
  <main class="pt-5">
    <div class="grid grid-cols-12 gap-8 mb-4 font-nathejk">

      <div class="col-span-12 md:col-span-6 xl:col-span-3">
          <div class="card !p-0 overflow-hidden flex flex-col">
              <div class="flex items-center p-4">
                  <i class="pi pi-users !text-5xl text-blue-500"></i>
                  <div class="ml-4">
                      <span class="text-blue-500 block whitespace-nowrap uppercase">Tilmeldte patruljer</span>
                      <span class="text-blue-500 block text-4xl font-bold">{{ config.patruljeCount }}</span>
                  </div>
              </div>
              <img src="/dashboard/users.svg" class="w-full" alt="users">
          </div>
      </div>

      <div class="col-span-12 md:col-span-6 xl:col-span-3">
          <div class="card !p-0 overflow-hidden flex flex-col">
              <div class="flex items-center p-4">
                  <i class="pi pi-face-smile !text-5xl text-orange-500"></i>
                  <div class="ml-4">
                      <span class="text-orange-500 block whitespace-nowrap">Tilmeldte gøglere</span>
                      <span class="text-orange-500 block text-4xl font-bold">{{ config.badutCount }}</span>
                  </div>
              </div>
              <img src="/dashboard/locations.svg" class="w-full" alt="locations">
          </div>
      </div>

      <div class="col-span-12 md:col-span-6 xl:col-span-3"><div class="card !p-0 overflow-hidden flex flex-col"><div class="flex items-center p-4"><i class="pi pi-directions !text-5xl text-green-500"></i><div class="ml-4"><span class="text-green-500 block whitespace-nowrap">CONVERSION RATE</span><span class="text-green-500 block text-4xl font-bold">12.6%</span></div></div><img src="/dashboard/rate.svg" class="w-full" alt="conversion"></div></div>

      <div class="col-span-12 md:col-span-6 xl:col-span-3"><div class="card h-full !p-0 overflow-hidden flex flex-col"><div class="flex items-center p-4"><i class="pi pi-comments !text-5xl text-purple-500"></i><div class="ml-4"><span class="text-purple-500 block whitespace-nowrap">ACTIVE TRIALS</span><span class="text-purple-500 block text-4xl font-bold">440</span></div></div><img src="/dashboard/interactions.svg" class="w-full mt-auto" alt="interactions"></div></div>

      <!--
      <div class="col-span-12 xl:col-span-6"><div class="card"><div class="font-semibold text-xl mb-4">Monthly Recurring Revenue Growth</div><div class="p-chart" id="nasdaq-chart" data-pc-name="chart" pc73="" data-pc-section="root" style="position: relative;"><canvas width="1232" height="740" data-pc-section="canvas" style="display: block; box-sizing: border-box; height: 370px; width: 616px;"></canvas></div></div></div>
      -->
      </div>
    <!--TheWelcome /-->
  </main>
</template>

<style>
.card:last-child {
    margin-bottom: 0;
}
.card {
    background: white;
    padding: 1.5rem;
    margin-bottom: 1rem;
    box-shadow: 0 3px 4px #0000001a, 0 24px 36px #0000000a;
    border-radius: 14px;
}
.card:hover {
    box-shadow: 0 3px 4px #000000cc, 0 24px 36px #000000cc;
}
</style>
