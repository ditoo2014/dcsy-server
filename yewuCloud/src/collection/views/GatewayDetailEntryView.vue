<script setup>
import { computed, shallowRef, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { resolveGatewayDetailSlugParam } from '../config/gateway-detail';
const route = useRoute();
const router = useRouter();
const modelSlug = computed(() => String(route.params.modelSlug || ''));
const loaders = {
    ttdc: () => import('./GatewayDetailView.vue'),
};
const Resolved = shallowRef(null);
const unsupported = ref(false);
let loadId = 0;
watch(modelSlug, slug => {
    const currentLoadId = ++loadId;
    Resolved.value = null;
    unsupported.value = false;
    const canonical = resolveGatewayDetailSlugParam(slug);
    if (!canonical) {
        unsupported.value = true;
        return;
    }
    const load = loaders[canonical];
    if (!load) {
        unsupported.value = true;
        return;
    }
    void load()
        .then(module => {
        if (currentLoadId !== loadId)
            return;
        Resolved.value = module.default;
    })
        .catch(() => {
        if (currentLoadId !== loadId)
            return;
        unsupported.value = true;
    });
}, { immediate: true });
function backToList() {
    void router.push('/collection');
}
</script>

<template>
  <component :is="Resolved" v-if="Resolved" />
  <main v-else-if="unsupported" class="unknown-slug">
    <el-empty description="该机型暂无详情页或链接不正确">
      <el-button type="primary" @click="backToList">返回首页</el-button>
    </el-empty>
  </main>
</template>

<style scoped>
.unknown-slug {
  min-height: 100%;
  padding: 48px 24px;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  background: #f0f2f5;
  box-sizing: border-box;
}
</style>
