<script setup lang="ts">
import { RouterLink } from "vue-router";
</script>

<template>
  <section class="not-found">
    <h1>404</h1>
    <p>The entity you are looking for has been sacrificed to the Entity.</p>
    <pre class="hook">
   .-'---`-.
 ,'          `.
 |             \
 |              \
 \           _  \
 ,\ _    ,'-,/-)\
 ( * \ \,' ,' ,'-)
  `._,)     -',-')
    \/         ''/
     )        / /
    /       ,'-'
   </pre>
    <p><RouterLink to="/">Back to the campfire</RouterLink></p>
  </section>
</template>

<style scoped>
.not-found { text-align: center; padding: 4rem 1rem; }
.not-found h1 { font-size: 4rem; margin: 0; color: var(--p-primary-color); opacity: 0.6; }
.not-found p { color: var(--p-text-muted-color); margin: 1rem 0; }
.hook { font-size: 0.6rem; line-height: 1.15; color: var(--p-text-muted-color); opacity: 0.5; margin: 2rem 0; user-select: none; }
</style>
