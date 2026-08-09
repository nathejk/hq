<script setup lang="ts">
import { computed, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import { http } from '@/plugins/axios';
import { useLiveResource } from '@/composables/useLiveResource';

const toast = useToast();

type HomeConfig = {
  timeCountdown?: string;
  videos?: unknown;
  patruljeCount: number;
  spejderCount: number;
  badutCount: number;
};

const emptyConfig: HomeConfig = { patruljeCount: 0, spejderCount: 0, badutCount: 0 };

// Every figure here is a derived count, which is precisely why dependsOn must name
// entity *types*: a count has no id for a signal to match, and a newly signed-up
// patrol carries an id this client has never seen. The handler (cmd/api/home.go)
// counts patrols with paidAmount > 0 and their members, plus paid gøglere — hence
// patrulje, spejder, gøgler, and order/payment for the paid part.
const { data, error } = useLiveResource<HomeConfig>(
  'home:config',
  async () => {
    const response = await http.get('/home');
    return (response.data.config ?? emptyConfig) as HomeConfig;
  },
  { dependsOn: ['patrulje', 'spejder', 'gøgler', 'order', 'payment'] },
);

const config = computed<HomeConfig>(() => data.value ?? emptyConfig);

watch(error, (err) => {
  if (!err) return;
  console.log('home load failed', err);
  toast.add({ severity: 'error', summary: 'Kunne ikke hente forsiden', life: 5000 });
});
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
                      <span class="text-orange-500 block whitespace-nowrap uppercase">Tilmeldte gøglere</span>
                      <span class="text-orange-500 block text-4xl font-bold">{{ config.badutCount }}</span>
                  </div>
              </div>
              <img src="/dashboard/locations.svg" class="w-full" alt="locations">
          </div>
      </div>

      <div class="col-span-12 md:col-span-6 xl:col-span-3">
          <div class="card !p-0 overflow-hidden flex flex-col">
              <div class="flex items-center p-4">
                  <i class="pi pi-qrcode !text-5xl text-green-500"></i>
                  <div class="ml-4">
                      <span class="text-green-500 block whitespace-nowrap uppercase">Antal fangster</span>
                      <span class="text-green-500 block text-4xl font-bold">0</span>
                  </div>
              </div>
              <img src="/dashboard/rate.svg" class="w-full" alt="conversion">
          </div>
      </div>

      <div class="col-span-12 md:col-span-6 xl:col-span-3">
          <div class="card h-full !p-0 overflow-hidden flex flex-col">
              <div class="flex items-center p-4">
                  <i class="pi pi-comments !text-5xl text-purple-500"></i>
                  <div class="ml-4">
                      <span class="text-purple-500 block whitespace-nowrap uppercase">Opkald til nødtelefon</span>
                      <span class="text-purple-500 block text-4xl font-bold">0</span>
                  </div>
              </div>
              <img src="/dashboard/interactions.svg" class="w-full mt-auto" alt="interactions">
          </div>
      </div>

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
