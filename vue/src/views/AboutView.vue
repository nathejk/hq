<script setup>
import { ref, onMounted } from 'vue';
//import { NodeService } from './service/NodeService';

const nodes = ref([{
    key: '0',
    label: 'Documents',
    data: 'Documents Folder',
    icon: 'pi pi-fw pi-inbox',
    children: [
        {
            key: '0-0',
            label: 'Work',
            data: 'Work Folder',
            icon: 'pi pi-fw pi-cog',
            children: [
                { key: '0-0-0', label: 'Expenses.doc', icon: 'pi pi-fw pi-file', data: 'Expenses Document' },
                { key: '0-0-1', label: 'Resume.doc', icon: 'pi pi-fw pi-file', data: 'Resume Document' }
            ]
        },
        {
            key: '0-1',
            label: 'Home',
            data: 'Home Folder',
            icon: 'pi pi-fw pi-home',
            children: [{ key: '0-1-0', label: 'Invoices.txt', icon: 'pi pi-fw pi-file', data: 'Invoices for this month' }]
        }
    ]
}]);
const selectedValue = ref(null);

onMounted(() => {
//    NodeService.getTreeNodes().then((data) => (nodes.value = data));
});
</script>

<template>
    <h1 class="font-nathejk text-2xl">Hjælpere</h1>
    <div class="card flex justify-center">
        <TreeSelect v-model="selectedValue" :options="nodes" display="chip"  selectionMode="checkbox" placeholder="Select Item" class="md:w-80 w-full" />
    </div>

  <div class="about">
    <h1>This is an about page</h1>
    <p>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    hello<br>
    </p>
  </div>
</template>

<style>
@media (min-width: 1024px) {
  .about {
    min-height: 100vh;
    display: flex;
    align-items: center;
  }
}
</style>
