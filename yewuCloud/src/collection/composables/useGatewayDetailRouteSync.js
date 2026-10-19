import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { gatewayDetailPath, modelToDetailSlug, pickGatewayModelField, resolveGatewayDetailSlugParam } from '../config/gatewayDetail'

/** 详情页在接口返回后校正 URL 机型段，避免详情页与网关实际机型不一致。 */
export function useGatewayDetailRouteSync() {
  const route = useRoute()
  const router = useRouter()

  function abortIfRouteMismatch(apiModelOrOne, gatewayId) {
    const urlParam = String(route.params.modelSlug || '').trim()
    const urlCanonical = resolveGatewayDetailSlugParam(urlParam)
    const raw = typeof apiModelOrOne === 'string' ? apiModelOrOne : pickGatewayModelField(apiModelOrOne)
    const slug = modelToDetailSlug(raw)

    // URL 已是支持的详情段但接口未返回机型时，以 URL 为准继续渲染。
    if (!slug && urlCanonical) return false
    if (!slug) {
      ElMessage.warning('该机型暂无详情页')
      void router.push('/collection')
      return true
    }
    if (!urlCanonical || urlCanonical !== slug) {
      const path = gatewayDetailPath(raw, gatewayId)
      if (path) void router.replace({ path, query: route.query })
      else void router.push('/collection')
      return true
    }
    return false
  }

  return { abortIfRouteMismatch }
}
