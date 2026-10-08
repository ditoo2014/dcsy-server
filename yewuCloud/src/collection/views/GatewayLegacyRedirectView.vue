<script setup>
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { useAuthStore } from '../stores/auth';
import { fetchGatewayOne } from '../api/gateway-module';
import { gatewayDetailPath, normalizeGatewayModelString, pickGatewayModelField } from '../config/gateway-detail';
const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const loading = ref(true);
const gatewayId = computed(() => String(route.params.gatewayId || ''));
onMounted(() => {
    void (async () => {
        if (!authStore.token || !gatewayId.value) {
            loading.value = false;
            void router.replace('/collection');
            return;
        }
        try {
            const { data } = await fetchGatewayOne({
                token: authStore.token,
                gatewayId: gatewayId.value,
            });
            if (data?.code !== 200) {
                ElMessage.error(data?.msg || '查询网关失败');
                void router.replace('/collection');
                return;
            }
            const one = data?.return?.data ?? {};
            const model = normalizeGatewayModelString(pickGatewayModelField(one));
            const path = gatewayDetailPath(model, gatewayId.value);
            if (!path) {
                ElMessage.warning('该机型暂无详情页');
                void router.replace('/collection');
                return;
            }
            void router.replace({ path, query: route.query });
        }
        catch (error) {
            ElMessage.error(error?.response?.data?.msg || '网关请求失败');
            void router.replace('/collection');
        }
        finally {
            loading.value = false;
        }
    })();
});
</script>

<template>
  <div class="legacy-redirect" v-loading="loading" element-loading-text="正在进入对应机型详情…" />
</template>

<style scoped>
.legacy-redirect {
  min-height: 200px;
}
</style>
