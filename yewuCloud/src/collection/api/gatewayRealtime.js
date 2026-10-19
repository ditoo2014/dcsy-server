import axios from 'axios'
import { resolveBatteryBranchGatewayId } from '../config/gatewayDetail'
import { extractRealtimeItems } from '../utils/gatewayTimesequenceResponse'
import { createAuthHeaders } from '../utils/request'
import { attachAuthFailureRedirectInterceptor } from './authFailureRedirect'
export const gatewayRealtimeApi = axios.create({
  baseURL: import.meta.env.VITE_AUTH_BASE_URL,
  timeout: Number(import.meta.env.VITE_API_TIMEOUT || 15000)
})
attachAuthFailureRedirectInterceptor(gatewayRealtimeApi)
export async function fetchGatewayRealtime(params) {
  const { token, gatewayId, dataIds } = params
  return gatewayRealtimeApi.post('/tolink/timesequence/getdata', dataIds, {
    params: {
      GatewayID: gatewayId,
      Function: 'realtimedata',
      Formula: params.formula ?? 'undefined'
    },
    headers: createAuthHeaders(token)
  })
}
export async function fetchGatewayTimesequenceIntact(params) {
  const { token, gatewayId } = params
  const query = {
    GatewayID: gatewayId,
    Function: 'intact'
  }
  if (params.formula != null) { query.Formula = params.formula }
  return gatewayRealtimeApi.post('/tolink/timesequence/getdata', {}, {
    params: query,
    headers: createAuthHeaders(token)
  })
}
export async function fetchDcFsuServerCsv16Realtime(params) {
  const chunkSize = Math.max(1, Math.floor(params.chunkSize ?? 100))
  const allIds = [...params.dataIds]
    .map(id => String(id ?? '').trim())
    .filter(Boolean)
  const primary = String(params.primaryGatewayId ?? '').trim()
  const target = String(params.targetGatewayId ?? '').trim()
  const apiGatewayId = target && target !== primary ? resolveBatteryBranchGatewayId(primary) : primary
  const byId = new Map()
  const pointsOrder = []
  for (let i = 0; i < allIds.length; i += chunkSize) {
    const batch = allIds.slice(i, i + chunkSize)
    const { data } = await fetchGatewayRealtime({
      token: params.token,
      gatewayId: apiGatewayId,
      formula: params.formula ?? 'undefined',
      dataIds: batch
    })
    if (data?.code != null && data.code !== 200) {
      throw new Error(typeof data?.msg === 'string' ? data.msg : 'DC-FSU 实时数据请求失败')
    }
    const items = extractRealtimeItems(data)
    for (const item of items) {
      if (item.dataId) { byId.set(item.dataId, item) }
    }
    const pts = data?.return?.points
    if (Array.isArray(pts)) {
      for (const p of pts) {
        const point = String(p ?? '').trim()
        if (point && !pointsOrder.includes(point)) { pointsOrder.push(point) }
      }
    }
  }
  return { items: [...byId.values()], pointsOrder }
}
export async function fetchGatewaySequenceData(params) {
  const { token, gatewayId, functionName, dataId } = params
  const q = {
    GatewayID: gatewayId,
    Function: functionName
  }
  if (params.formula != null) { q.Formula = params.formula }
  return gatewayRealtimeApi.post('/tolink/timesequence/getdata', { dataId }, {
    params: q,
    headers: createAuthHeaders(token)
  })
}
export async function fetchGatewayHistoryPaged(params) {
  const q = {
    GatewayID: params.gatewayId,
    Function: 'historydata',
    Formula: params.formula ?? '',
    Pagenum: String(params.pageNo),
    Pagesize: String(params.pageSize),
    Datestart: String(params.dateStart),
    Dateend: String(params.dateEnd)
  }
  const response = await gatewayRealtimeApi.post('/tolink/timesequence/getdata', { dataId: params.dataId }, {
    params: q,
    headers: createAuthHeaders(params.token)
  })
  return checkHistoryResponse(response)
}
export async function fetchGatewayHistoryCount(params) {
  const q = {
    GatewayID: params.gatewayId,
    Function: 'countdata',
    Datestart: String(params.dateStart),
    Dateend: String(params.dateEnd)
  }
  const response = await gatewayRealtimeApi.post('/tolink/timesequence/getdata', { dataId: params.dataId }, {
    params: q,
    headers: createAuthHeaders(params.token)
  })
  return checkHistoryResponse(response)
}

// 时序接口失败时也可能返回 HTTP 200，必须检查业务状态，不能当作空历史。
function checkHistoryResponse(response) {
  if (Number(response.data?.code) !== 200) {
    throw new Error(response.data?.msg || '历史数据查询失败，请稍后重试')
  }
  return response
}
