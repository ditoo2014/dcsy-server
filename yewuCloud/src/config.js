import { PLATFORM } from './constants/platform'

const platform = import.meta.env.VITE_PLATFORM || PLATFORM.DCSY_XC
const ip = import.meta.env.VITE_API_HOST || window.location.hostname
const port = import.meta.env.VITE_API_PORT || window.location.port || '443'
let collectionBaseUrl = ''
let cloudConfigurationBaseUrl = ''
let cloudConfigurationPort = ''
if (platform === PLATFORM.DCSY_INTRANET) {
  if (port === '8003' || port === '80') { // 电池溯源
    cloudConfigurationPort = 8001
  } else if (port === '5002') { // 空港演示
    cloudConfigurationPort = 5001
  }
  cloudConfigurationBaseUrl = `http://${ip}:${cloudConfigurationPort}`
  collectionBaseUrl = `http://${ip}:81`
} else if (platform === PLATFORM.DCSY_XC) { // 信创
  cloudConfigurationPort = '8001'
  cloudConfigurationBaseUrl = `http://${ip}:${cloudConfigurationPort}`
  collectionBaseUrl = import.meta.env.VITE_COLLECTION_URL || '/dcsjgl/wgpz'
}

const config = {
  platform,
  ip, // 服务器ip
  port, // 业务云业务端口
  CJYDomainName: collectionBaseUrl, // 采集云域名
  YZTDomainName: cloudConfigurationBaseUrl, // 云组态
  JuriPort: Number(import.meta.env.VITE_AUTH_PORT || 9008) // 业务云权限端口
}

export default config
