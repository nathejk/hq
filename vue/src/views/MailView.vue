<script setup>
import { ref, onMounted } from 'vue';
import { PrimeIcons } from '@primevue/core/api';

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
//const selectedValue = ref(null);

onMounted(() => {
//    NodeService.getTreeNodes().then((data) => (nodes.value = data));
});

const items = ref([
    {
        label: 'Mails',
        items: [
            {
                label: 'Skriv ny',
                icon: PrimeIcons.PENCIL,
                route: '/mail/new'
            },
            {
                label: 'Udbakke',
                icon: PrimeIcons.ENVELOPE,
                route: '/mail/outbox'
            },
            {
                label: 'Skabeloner',
                icon: PrimeIcons.FOLDER_OPEN,
                route: '/mail/templates'
            }
        ]
    },
    {
        label: 'Profile',
        items: [
            {
                label: 'Settings',
                icon: 'pi pi-cog'
            },
            {
                label: 'Logout',
                icon: 'pi pi-sign-out'
            }
        ]
    }
]);
const selectedValue = ref(null);
/*const nodes = ref([{
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
}])*/
const mail = ref({sender:0});
const selectedSender = ref(0);
const senders = ref([
    { name: 'Nathejk <tilmeld@nathejk.dk>', code: 0 },
]);
</script>

<template>
    <h1 class="font-nathejk text-2xl py-3">Kommunikation <span>: <span class="text-slate-400">Skriv ny</span></span></h1>
    <div class="grid grid-cols-6 gap-4 mb-3">
        <div class="card flex justify-center">
            <Menu :model="items">
                <template #item="{ item, props }">
                <router-link v-if="item.route" v-slot="{ href, navigate }" :to="item.route" custom>
                    <a v-ripple :href="href" v-bind="props.action" @click="navigate">
                        <span :class="item.icon" />
                        <span class="ml-2">{{ item.label }}</span>
                    </a>
                </router-link>
            </template>
            </Menu>
        </div>
        <div class="col-span-5 ">
                <div class="card ">
                    <div class="grid grid-cols-3 gap-4">

                    <div class="col-span-2">
                        <FloatLabel variant="on">
                            <TreeSelect v-model="mail.recipients" id="username" :options="nodes" selectionMode="checkbox" class="w-full" fluid />
                            <label for="username">Modtagere</label>
                        </FloatLabel>
                    </div>
                    <div class="col-span-2">
                        <FloatLabel variant="on">
                            <Select v-model="mail.sender" id="sender" :options="senders" optionLabel="name" optionValue="code" class="w-full " />
                            <label for="sender">Afsender</label>
                        </FloatLabel>
                    </div>
                    <div class="col-span-2">
                        <FloatLabel variant="on">
                            <InputText type="text" v-model="mail.subject" id="subject" class="w-full" />
                            <label for="subject">Emne</label>
                        </FloatLabel>
                    </div>
                    <div class="col-span-2">
                        <Editor v-model="mail.body" editorStyle="height: 320px" fluid >
                            <template v-slot:toolbar>
                                <span class="ql-formats">
                                    <button v-tooltip.bottom="'Bold'" class="ql-bold"></button>
                                    <button v-tooltip.bottom="'Italic'" class="ql-italic"></button>
                                    <button v-tooltip.bottom="'Underline'" class="ql-underline"></button>
                                </span>
                            </template>
                        </Editor>
                    </div>
                    <div class="bg-yellow-100">
                        variable:
                        #TEAM#

                    </div>
                    <div class="col-span-2">
                        <Button class="float-right" label="Send e-mail" icon="pi pi-user" />

                    </div>
                    </div>
                </div>
        </div>
    </div>


</template>

<style>
</style>
