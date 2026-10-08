/** 部署环境标识（构建时注入，避免在公网包中硬编码内网 IP） */
export const PLATFORM = {
  DCSY_INTRANET: 'dcsy-intranet',
  DCSY_XC: 'dcsy-xc'
}

export function isDcsyLoginUi(platform) {
  return platform === PLATFORM.DCSY_XC
}

export function isDcsyMainUi(platform) {
  return platform === PLATFORM.DCSY_XC ||
    platform === PLATFORM.DCSY_INTRANET
}

export function isDcsyTracePlatform(platform, port) {
  if (platform === PLATFORM.DCSY_XC) {
    return true
  }
  if (platform === PLATFORM.DCSY_INTRANET) {
    return port === '8003' || port === '80'
  }
  return false
}

export function getYztUserid(config) {
  if (config.platform === PLATFORM.DCSY_INTRANET) {
    if (config.port === '8003' || config.port === '80') {
      return `${config.ip}:8004`
    }
    if (config.port === '5002') {
      return `${config.ip}:5000`
    }
  }
  if (config.platform === PLATFORM.DCSY_XC) {
    return `${config.ip}:8004`
  }
  return config.ip
}

