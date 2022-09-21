<template>
  <header>
    <nav class="navbar navbar-expand-md fixed-top bg-dark">
      <div class="container-fluid h-100 p-0">
        <a class="navbar-brand" href="./"><i class="fas fa-moon fa-flip-vertical" :class="[statusClass]"></i> Nathejk</a>
        <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#navbarCollapse" aria-controls="navbarCollapse" aria-expanded="false" aria-label="Toggle navigation">
          <span class="navbar-toggler-icon"></span>
        </button>
        <div class="collapse navbar-collapse" id="navbarCollapse">
            <ul class="navbar-nav me-auto mb-2 mb-lg-0 nav nav-tabs">
              <li class="nav-item">
                <router-link :to="{name:'home'}" class="nav-link mb-0" active-class="active" exact>
                  <i class="fas fa-lg fa-fw fa-home"></i><small>Hjem</small>
                </router-link>
              </li>
              <li class="nav-item" v-for="(m, i) in menu" :key="i">
                  <a v-if="m.link.substring(0, 4) == 'http'" class="nav-link mb-0" :href="m.link">
                    <i class="fas fa-lg fa-fw" :class="m.icon"></i><small>{{ m.label }}</small> {{ m.text}}
                  </a>
                  <router-link v-else :to="m.link" class="nav-link mb-0" active-class="active" __exact>
                    <i class="fas fa-lg fa-fw" :class="m.icon"></i><small>{{ m.label }}</small> {{ m.text}}
                  </router-link>
              </li>
            </ul>
          <ul class="navbar-nav ml-auto mb-2 mb-lg-0" old="navbar-nav ml-auto navr-right">
            <li class="nav-item dropdown">
              <a class="nav-link dropdown-toggle" href="http://example.com" id="dropdown01" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false"><i class="far fa-user"></i> {{ user.name }}</a>
              <div class="dropdown-menu dropdown-menu-right" aria-labelledby="dropdown01">
                <a class="dropdown-item" href="http://lukmigind.nathejk.dk/logout"><i class="fas fa-sign-out-alt fa-fw"></i> Log ud</a>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </nav>
  </header>
</template>

<style lang="scss">
header {
    padding-top:60px;
}
nav.navbar {
    height: 60px; 
    padding:0 1rem;
    background-image: linear-gradient(to bottom,#445e65 0,#1f2a26 100%);
    background-repeat: repeat-x;
}
nav a.navbar-brand {
    font: bold 30px/60px 'helvetica neue',helvetica,arial,sans-serif;
    text-transform: uppercase;
    color: white;

    i {
        font-size:1.2rem;
        vertical-align: text-top;
    }
}
nav .navbar-collapse { padding-top:.5rem;
  align-self: flex-end;
}
nav ul.nav { border-bottom: 0;
  align-self: flex-end;
}
nav ul.nav li.nav-item {margin-right:.2rem; margin-bottom:0;}
nav ul.nav li.nav-item a {
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
}
nav li.nav-item a {
    color:rgba(255,255,255,.5);
}
nav ul.nav li.nav-item a.active {
    background:#fff;
    
    small {
        display:inline;
    }
}
nav ul.nav li.nav-item a small {
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

@media (min-width: 992px) {
  .modal-lg {
    max-width: 80%;
  }
}
</style>

<script>
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
            //{icon:'fa-search', label:'Søg', link:'/search'},
            {icon:'fa-headset', label:'Nødtelefon', link:'/sos'},
            {icon:'fa-user-injured', label:'Udgået', link:'/ude'},
            {icon:'fa-fist-raised', label:'LOK', link:'/lok'},
            {icon:'fa-child', text:'Patruljer', link:'/patruljer'},
            {icon:'fa-search', text:'Søg', link:'http://natpas.nathejk.dk/search'},
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
