import defaultSettings from '@/settings'
import config from '@/config.js'
import { PLATFORM } from '@/constants/platform'

let title = defaultSettings.title || 'Vue Element Admin'
switch (config.platform) {
  case PLATFORM.DCSY_INTRANET:
    if (config.port === '8003' || config.port === '80') {
      title = '锂硫电池溯源平台'
    } else if (config.port === '5002') {
      title = '物联网云平台'
    }
    break
  case PLATFORM.DCSY_XC:
    title = '锂硫电池溯源平台'
    break
}

export default function getPageTitle(pageTitle) {
  if (pageTitle) {
    return `${pageTitle} - ${title}`
  }
  return `${title}`
}
