<template>
  <header>

    <b-navbar toggleable="lg" type="dark" fixed="top">

      <a class="navbar-brand" href="./"><i class="fas fa-moon fa-flip-vertical" :class="[statusClass]"></i> Nathejk</a>
      <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>

      <b-collapse id="nav-collapse" is-nav>
        <b-navbar-nav class="tabs">
          <b-nav-item :to="{name:'home'}" active-class="active" exact>
            <i class="fas fa-fw fa-home"></i><small>Hjem</small>
          </b-nav-item>
          <b-nav-item v-for="(m, i) in menu" :key="i" :to="m.link" :href="m.href" active-class="active">
            <i class="fas fa-fw" :class="m.icon"></i><small>{{ m.label }}</small>{{ m.text}}
          </b-nav-item>
        </b-navbar-nav>

        <b-navbar-nav class="ml-auto">
          <b-nav-item-dropdown right>
            <template #button-content>
              <i class="far fa-user fa-fw"></i> {{ user.name }}
            </template>
            <b-dropdown-item :to="{ name: 'years' }"><i class="far fa-calendar-times fa-fw"></i> Udgaver</b-dropdown-item>
                <b-dropdown-divider></b-dropdown-divider>
            <b-dropdown-item :to="{ name: 'years' }"><i class="far fa-file fa-fw"></i> Filer <i class="fas fa-external-link-alt fa-xs"></i></b-dropdown-item>
            <b-dropdown-item :to="{ name: 'years' }"><i class="fas fa-users fa-fw"></i> Tilmelding <i class="fas fa-external-link-alt fa-xs"></i></b-dropdown-item>

                    <b-dropdown-divider></b-dropdown-divider>

            <b-dropdown-item href="http://lukmigind.nathejk.dk/logout"><i class="fas fa-sign-out-alt fa-fw"></i> Log ud</b-dropdown-item>
          </b-nav-item-dropdown>
        </b-navbar-nav>
      </b-collapse>

    </b-navbar>

  </header>
</template>

<style lang="scss">
header {
    padding-top:60px;
}
nav.navbar {
    min-height: 60px;
    padding:0 1rem;
    background-image: linear-gradient(to bottom,#445e65 0,#1f2a26 100%);
    background-repeat: repeat-x;
}
nav a.navbar-brand {
    font: bold 30px/60px 'helvetica neue',helvetica,arial,sans-serif;
    text-transform: uppercase;
    color: white;
    padding:0;

    i {
        font-size:1.2rem;
        vertical-align: text-top;
    }
}
nav .navbar-collapse {
    padding-top:.5rem;
    align-self: flex-end;
}
nav ul { border-bottom: 0;
    align-self: flex-end;
}
#nav-collapse:not(.show) .tabs li.nav-item {

    margin-right:.2rem;
    margin-bottom:0;

    a {
        /*background-image: linear-gradient(to bottom,#1f2a26 0,#445e65 100%);*/
        background-color:#445e65;
        background-repeat: repeat-x;
        border-top-left-radius:.5rem;
        border-top-right-radius:.5rem; 
        color:rgba(255,255,255,.5);
        padding:.25rem .5rem;

        &:hover small {
            display:inline;
        }
        i {
            font-size: 1.33333em;
            line-height: .75em;
            vertical-align: -0.0667em;
        }
        small {
            position: absolute;
            width: 58px;
            text-align: center;
            margin-left: -42px;
            font-size: 8px;
            margin-top: 28px;
            text-transform: uppercase;
            display:none;
            color:#445e65;
        }
    }
    a.active {
        background:#fff;
        color:#445e65;
        small {
            display:inline;
        }
    }
}
#nav-collapse.show li.nav-item small {
    padding-left:10px;
}
@media (min-width: 992px) {
  .modal-lg {
    max-width: 80%;
  }
}
unused {
}
</style>

<script>
import { BNavbar, BCollapse, BNavbarToggle, BNavItemDropdown, BDropdownItem } from 'bootstrap-vue'
export default {
    props: {
        title: null
    },
    components: { BNavbar, BCollapse, BNavbarToggle, BNavItemDropdown, BDropdownItem },

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
}
</script>
