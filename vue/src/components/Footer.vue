<template>
        <footer v-if="show" class=" sticky top-full bottom-0 w-full ">
            <div class="container mx-auto pt-5">
                <div class="grid grid-cols-12 gap-4">
                    <div class="col-start-2">
                            <img src="/img/logomoonsmall.png" alt="Nathejk" width="59" height="40" class="mr-3 mt-1">
                    </div>
                    <div class="col-span-4">
                        <div class="media">
                            <div class="media-body">
                                <p>Nathejk {{ year }}<br><small>Vi ses i mørket...</small></p>
                                <p class="mb-3">
                                    <a href="https://facebook.com/nathejk" target="_blank" rel="nofollow"><i class="fab fa-facebook-square mr-1"></i></a>
                                    <a href="https://www.instagram.com/nathejk/" target="_blank" rel="nofollow"><i class="fab fa-instagram"></i></a>
                                </p>
                                <p class="grey mb-0">
                                <small>2011-{{ year }} Nathejk.<br>Bygget i halvmånens skær <small class="yellow">&#9790;</small> et sted på Sjælland.</small>
                                </p>
                                <br class="d-md-none">
                            </div>
                        </div>
                    </div>
                    <div class="col-span-3">
                        <ul class="list-unstyled">
                            <li><small><a href="http://nathejk.dk">Nathejk</a></small></li>
                            <li><small><a href="https://tilmelding.nathejk.dk">Nathejk tilmelding</a></small></li>
                            <li><small><a href="https://status.nathejk.dk">Nathejk status</a></small></li>
                            <li><small><a href="https://tilmelding.nathejk.dk/diplom">Patrulje diplomer</a></small></li>
                        </ul>
                    </div>
                    <div class="col-span-3">
                        <ul class="list-unstyled">
                            <li><small><a href="https://dds.dk">Det Danske Spejderkorps</a></small></li>
                            <li><small><a href="http://adventurespejd.dk/forside/">Adventurespejd liga</a></small></li>
                        </ul>
                    </div>
                </div>
            </div>
        </footer>
</template>

<style>
footer {
    background-color: rgb(50,67,68);
    color: rgba(255, 255, 255, 0.5);
    font-size: .9rem;

    a {
        color: rgba(255, 255, 255, 0.5);
        &:hover {
            color: white;
        }
    }
    .media {
        a, p { color:#eee; }
        p.grey { color: rgba(255, 255, 255, 0.5); }

        .yellow {
            color:yellow;
            transform: rotate(60deg);
            display:inline-block;
        }
    }
}
</style>

<script>
export default {
    props: {
        title: null
    },
    computed: {
        year() { return new Date().getFullYear() },
        show() { return !('footer' in this.$route.meta) || this.$route.meta.footer }
    },

}
</script>
