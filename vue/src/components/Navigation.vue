<script setup>
    import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

//import { fas, far, fal, fass, fasds } from '@awesome.me/kit-KIT_CODE/icons'
//import { faMoon, faLock, faWarning } from '@fortawesome/free-solid-svg-icons'

const props = defineProps({
    title: String,
    class: String,
})
/*<{
     title: string,
     class: string,
   }>();
*/
    import { ref } from "vue";
const onoff = ref(false)
const items = ref([
    {
        label: 'Home',
        icon: 'fas fa-home',
        name: 'home',
    },
    {
        label: 'Banditter',
        icon: 'fas fa-user-secret',
        name: 'klaner',
    },
    {
        label: 'Spejdere',
        icon: 'fas fa-users',
        name: 'patruljer',
    },
    {
        label: 'Gøgl',
        icon: 'fas fa-masks-theater',
        name:'badutter',
    },
    {
        label: 'Projects',
        icon: 'pi pi-search',
        items: [
            {
                label: 'Core',
                icon: 'pi pi-bolt',
                shortcut: '⌘+S'
            },
            {
                label: 'Blocks',
                icon: 'pi pi-server',
                shortcut: '⌘+B'
            },
            {
                label: 'UI Kit',
                icon: 'pi pi-pencil',
                shortcut: '⌘+U'
            },
            {
                separator: true
            },
            {
                label: 'Templates',
                icon: 'pi pi-palette',
                items: [
                    {
                        label: 'Apollo',
                        icon: 'pi pi-palette',
                        badge: 2
                    },
                    {
                        label: 'Ultima',
                        icon: 'pi pi-palette',
                        badge: 3
                    }
                ]
            }
        ]
    },
    {
        label: 'Contact',
        icon: 'pi pi-envelope',
        badge: 3
    }
]);
/*
export default {
    props: {
        title: null
    },

    data: () => ({
        activeMenu: 'søg',
        menu: [
            //{icon:'fa-home', label:'Hjem', link:'/'},
            {icon:'fa-sitemap', label:'Organisation', link:'/organisation'},
            {icon:'fa-map', label:'Kort', link:'/kort'},
            {icon:'fa-map-marker-alt', label:'Poster', link:'/poster'},
//            {icon:'fa-book-open', label:'Søg', link:'/hej'},
            {icon:'fa-headset', label:'Nødtelefon', link:'/sos'},
            {icon:'fa-user-injured', label:'Udgået', link:'/ude'},
//            {icon:'fa-fist-raised', label:'LOK', link:'/lok'},
            {icon:'fa-child', text:'Patruljer', link:'/patruljer'},
            {icon:'fa-search', text:'Søg', href:'http://natpas.nathejk.dk/search'},
        ],
    }),
    computed: {
      user() {
  return this.$store.getters['dims/user']
      },
      statusClass() {
        switch (this.$store.getters['dims/websocketStatus']) {
        case 'open': return 'text-warning'
        case 'closed': return 'text-dark'
        case 'errored': return 'text-danger'
        default: return 'text-light'
        }
      },
    },
    methods: {
        isActive: function(t) { return t == this.activeMenu },
        setActive: function (e) { this.activeMenu = e.currentTarget.id; },
        subIsActive(input) {
            const paths = Array.isArray(input) ? input : [input]
            return paths.some(path => {
                return this.$route.path.indexOf(path) === 0 // current path starts with this path string
            })
        },
    }
}*/
import 'primeicons/primeicons.css'
</script>
<template>

    <nav class="bg-gray-800 text-white shadow-md sticky top-0 w-full z-50">
        <div class="container mx-auto">
            <div class="flex justify-between items-center">
                <!-- Logo and Brand Name -->
      <a class="font-nathejk text-2xl leading-relaxed pr-5 uppercase" href="/"><FontAwesomeIcon :icon="['fas', 'moon']" flip="vertical" class="align-top text-yellow-400" />Nathejk</a>
                
                <!-- Navigation Icons -->
                <div class="flex">
                    <template v-for="item in items">
                        <router-link v-if="item.name" v-slot="{ isActive, href, navigate }" :to="{name: item.name}" custom>
                            <a v-ripple :href="href" v-bind="props.action" @click="navigate" :class="{'bg-gray-600 ':isActive}" class="group p-4 hover:bg-gray-700 border-r border-gray-700 relative grid place-items-center">
                                <i :class="item.icon" class="text-2xl" />
                                <span :class="{'opacity-0':!isActive}" class="group-hover:opacity-100 duration-300 text-xs block mt-1 absolute bottom-0 uppercase">{{ item.label }}</span>
                            </a>
                        </router-link>
                        <a v-else :href="item.url" :class="{'bg-gray-600 ':item.active}" class="group p-4 hover:bg-gray-700 border-r border-gray-700 relative grid place-items-center">
                            <i :class="item.icon" class="text-2xl" />
                            <span :class="{'opacity-0':!item.active}" class="group-hover:opacity-100 duration-300 text-xs block mt-1 absolute bottom-0 uppercase">{{ item.label }}</span>
                        </a>
                    </template>

<!--
                    <a href="#" class="p-4 hover:bg-gray-700 border-r border-gray-700 grid place-items-center">
                        <i class="fas fa-home text-xl"></i>
                    </a>
                    <a href="#" class="p-4 hover:bg-gray-700 border-r border-gray-700">
                        <i class="fas fa-users text-xl"></i>
                    </a>
                    <a href="#" class="p-4 hover:bg-gray-700 border-r border-gray-700">
                        <i class="fas fa-map text-xl"></i>
                    </a>
                    <a href="#" class="p-4 hover:bg-gray-700 border-r border-gray-700 bg-gray-600 relative grid place-items-center">
                        <i class="fas fa-map-marker-alt text-xl"></i>
                        <span class="text-xs block mt-1 absolute bottom-0">POSTER</span>
                    </a>
                    <a href="#" class="p-4 hover:bg-gray-700 border-r border-gray-700">
                        <i class="fas fa-headset text-xl"></i>
                    </a>
                    <a href="#" class="p-4 hover:bg-gray-700 border-r border-gray-700">
                        <i class="fas fa-user-circle text-xl"></i>
                    </a>
                    <a href="#" class="p-4 hover:bg-gray-700 border-r border-gray-700">
                        <i class="fas fa-clipboard-list text-xl"></i>
                        <span class="text-gray-300">Patruljer</span>
                    </a>
                    <a href="#" class="p-4 hover:bg-gray-700">
                        <i class="fas fa-search text-xl"></i>
                        <span class="text-gray-300">Søg</span>
                    </a>
-->
                </div>
                
                <!-- User Profile -->
                <div class="flex items-center p-2">
                    <span class="mr-2">nathejk</span>
                    <i class="fas fa-chevron-down"></i>
                </div>
            </div>
        </div>
    </nav>

        <Menubar v-if="onoff" :model="items" class="border-0 bg-transparent fixed top-0 w-full">
            <template #start>
      <a class="font-nathejk text-2xl leading-relaxed pr-5 uppercase" href="./"><FontAwesomeIcon :icon="['fas', 'moon']" flip="vertical" class="align-top text-yellow-400" />Nathejk</a>

            </template>
            <template #item="{ item, props, hasSubmenu, root }">
                <a v-ripple class="flex items-center" v-bind="props.action" style="color:#a2aeb3!important">
                    <span :class="item.icon" />
                    <span class="">{{ item.label }}</span>
                    <Badge v-if="item.badge" :class="{ 'ml-auto': !root, 'ml-2': root }" :value="item.badge" />
                    <span v-if="item.shortcut" class="ml-auto border border-surface rounded bg-emphasis text-muted-color text-xs p-1">{{ item.shortcut }}</span>
                    <i v-if="hasSubmenu" :class="['pi pi-angle-down', { 'pi-angle-down ml-2': root, 'pi-angle-right ml-auto': !root }]"></i>
                </a>
            </template>
            <template #end>
                <div class="flex items-center gap-2">
                    <!--InputText placeholder="Search" type="text" class="w-32 sm:w-auto" /-->
                    <Avatar image="https://primefaces.org/cdn/primevue/images/avatar/amyelsner.png" shape="circle" />
                    Brugernavn
                </div>
            </template>
        </Menubar>


</template>

<style lang="css">
header_ {
    padding-top:60px;
}
nav {
    border:0 ! important;
    background-image: linear-gradient(to bottom,#445e65 0,#1f2a26 100%) ! important;
}
.p-menubar-item {
    color: #a2aeb3!important;
}
.p-menubar-item-content:hover,
.p-focus .p-menubar-item-content   {
    background-color:#445e65!important;
}
</style>

