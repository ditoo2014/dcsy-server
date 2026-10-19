const PASSWORD_POLICY_MIN_LENGTH = 8
const PASSWORD_POLICY_MAX_LENGTH = 15

export const PASSWORD_POLICY_ERROR_CODE = 'PASSWORD_POLICY_VIOLATION'
export const PASSWORD_POLICY_MESSAGE = '密码必须包含大小写字母、数字、特殊符号，长度在8-15位；密码不能连续字符（如123、abc）、密码不能为相同字符（如111、aaa）连续3位或3位以上；数字非1234等规律排列，字母非人名、公司名等易猜测组合'

const PASSWORD_COMMON_WEAK_TOKENS = [
  'password',
  'passwd',
  'admin',
  'root',
  'user',
  'test',
  'qwerty',
  'qwe',
  'asdf',
  'zxcv',
  'welcome',
  'abc'
]

const PASSWORD_GUESSABLE_INFO_TOKENS = [
  'ditoo',
  'dicloud',
  'ywcloud',
  'yewucloud',
  'sctower',
  'sctel',
  'tower',
  'company',
  'corp',
  'platform',
  'zhangsan',
  'lisi',
  'wangwu',
  'zhaoliu',
  'zhangwei',
  'wangwei',
  'liwei',
  'zhaomin',
  'xiaoming',
  'xiaohong',
  'xiaoli',
  'xiaowang',
  'jack',
  'tom',
  'jerry',
  'mike',
  'david',
  'alice',
  'lucy',
  'lily',
  'mary',
  'john'
]

function toText(value) {
  return value == null ? '' : String(value)
}

function runeLength(value) {
  return Array.from(toText(value)).length
}

function hasUpperLowerDigitSpecial(value) {
  if (!/^[\x21-\x7E]+$/.test(toText(value))) {
    return false
  }

  let hasUpper = false
  let hasLower = false
  let hasDigit = false
  let hasSpecial = false

  for (const char of Array.from(toText(value))) {
    if (/[A-Z]/.test(char)) {
      hasUpper = true
      continue
    }
    if (/[a-z]/.test(char)) {
      hasLower = true
      continue
    }
    if (/[0-9]/.test(char)) {
      hasDigit = true
      continue
    }
    if (!/[A-Za-z0-9\u4e00-\u9fa5]/.test(char)) {
      hasSpecial = true
    }
  }

  return hasUpper && hasLower && hasDigit && hasSpecial
}

function hasSequentialCharacterRun(value, sequenceLength = 3) {
  const chars = Array.from(toText(value).toLowerCase())
  let runLength = 1
  let previousStep = 0

  for (let index = 1; index < chars.length; index++) {
    const previous = chars[index - 1]
    const current = chars[index]
    const bothLetters = /[a-z]/.test(previous) && /[a-z]/.test(current)
    const bothDigits = /[0-9]/.test(previous) && /[0-9]/.test(current)
    const step = current.charCodeAt(0) - previous.charCodeAt(0)

    if ((bothLetters || bothDigits) && (step === 1 || step === -1)) {
      if (runLength === 1 || step === previousStep) {
        runLength++
      } else {
        runLength = 2
      }
      previousStep = step
      if (runLength >= sequenceLength) {
        return true
      }
      continue
    }

    runLength = 1
    previousStep = 0
  }

  return false
}

function hasRepeatedRune(value, repeatCount) {
  const chars = Array.from(toText(value))
  if (chars.length < repeatCount) {
    return false
  }

  let count = 1
  for (let i = 1; i < chars.length; i++) {
    if (chars[i] === chars[i - 1]) {
      count++
      if (count >= repeatCount) {
        return true
      }
    } else {
      count = 1
    }
  }

  return false
}

function isArithmeticDigitSequence(digits) {
  if (digits.length < 3) {
    return false
  }

  const step = digits[1] - digits[0]
  if (step === 0) {
    return false
  }

  for (let i = 1; i < digits.length; i++) {
    if (digits[i] - digits[i - 1] !== step) {
      return false
    }
  }

  return true
}

function hasRepeatedDigitBlock(segment) {
  for (let blockSize = 1; blockSize <= 3; blockSize++) {
    if (segment.length < blockSize * 2 || segment.length % blockSize !== 0) {
      continue
    }
    const block = segment.slice(0, blockSize)
    let match = true
    for (let index = blockSize; index < segment.length; index += blockSize) {
      if (segment.slice(index, index + blockSize) !== block) {
        match = false
        break
      }
    }
    if (match) {
      return true
    }
  }
  return false
}

function hasRepeatedDigitRunPattern(digits) {
  if (digits.length < 3) {
    return false
  }

  const runs = []
  for (const digit of digits) {
    const last = runs[runs.length - 1]
    if (!last || last.value !== digit) {
      runs.push({ value: digit, count: 1 })
    } else {
      last.count += 1
    }
  }

  if (runs.length < 3) {
    return false
  }

  const baseCount = runs[0].count
  if (baseCount < 2) {
    return false
  }

  if (!runs.every(run => run.count === baseCount)) {
    return false
  }

  const values = runs.map(run => run.value)
  return isArithmeticDigitSequence(values)
}

function hasWeakDigitPattern(value) {
  const segments = toText(value).match(/\d+/g) || []
  for (const segment of segments) {
    const digits = segment.split('').map(Number)
    if (digits.length < 3) {
      continue
    }
    if (digits.every(digit => digit === digits[0])) {
      return true
    }
    if (isArithmeticDigitSequence(digits)) {
      return true
    }
    if (hasRepeatedDigitBlock(segment)) {
      return true
    }
    if (hasRepeatedDigitRunPattern(digits)) {
      return true
    }
  }

  return false
}

function dedupeTokens(tokens) {
  const seen = new Set()
  const result = []
  for (const token of tokens) {
    const clean = toText(token).trim()
    if (!clean) {
      continue
    }
    const key = clean.toLowerCase()
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    result.push(clean)
  }
  return result
}

function shouldCheckPasswordToken(token) {
  const chars = Array.from(toText(token).trim())
  if (chars.length >= 3) {
    return true
  }
  if (chars.length < 2) {
    return false
  }
  return chars.some(char => /[\u4e00-\u9fa5]/.test(char))
}

function collectContextTokens(context = {}) {
  const fields = [
    context.username,
    context.trueName,
    context.truename,
    context.email,
    context.phone,
    context.paleName,
    context.paleType,
    context.platName,
    context.platType,
    context.renter,
    context.clientId,
    context.corpName,
    context.companyName
  ]

  const tokens = []
  for (const field of fields) {
    const text = toText(field).trim()
    if (!text) {
      continue
    }
    tokens.push(text)
    if (text.includes('@')) {
      tokens.push(text.split('@')[0])
    }
    const asciiTokens = text.match(/[A-Za-z0-9]+/g) || []
    tokens.push(...asciiTokens)
  }

  const phone = toText(context.phone).trim()
  if (/^\d{11}$/.test(phone)) {
    tokens.push(phone.slice(0, 4))
    tokens.push(phone.slice(7))
  }

  return dedupeTokens(tokens)
}

function getPasswordPolicyMessage(value, context = {}) {
  const password = toText(value)
  if (!password) {
    return PASSWORD_POLICY_MESSAGE
  }

  const passwordLength = runeLength(password)
  if (passwordLength < PASSWORD_POLICY_MIN_LENGTH || passwordLength > PASSWORD_POLICY_MAX_LENGTH) {
    return PASSWORD_POLICY_MESSAGE
  }

  if (!hasUpperLowerDigitSpecial(password)) {
    return PASSWORD_POLICY_MESSAGE
  }

  if (hasRepeatedRune(password, 3)) {
    return PASSWORD_POLICY_MESSAGE
  }

  if (hasSequentialCharacterRun(password, 3)) {
    return PASSWORD_POLICY_MESSAGE
  }

  if (hasWeakDigitPattern(password)) {
    return PASSWORD_POLICY_MESSAGE
  }

  const passwordLower = password.toLowerCase()
  const weakTokens = collectContextTokens(context)
  for (const token of weakTokens) {
    if (!shouldCheckPasswordToken(token)) {
      continue
    }
    const tokenLower = token.toLowerCase()
    if (passwordLower.includes(tokenLower) || password.includes(token)) {
      return PASSWORD_POLICY_MESSAGE
    }
  }

  for (const token of PASSWORD_GUESSABLE_INFO_TOKENS) {
    if (passwordLower.includes(token)) {
      return PASSWORD_POLICY_MESSAGE
    }
  }

  for (const token of PASSWORD_COMMON_WEAK_TOKENS) {
    if (passwordLower.includes(token)) {
      return PASSWORD_POLICY_MESSAGE
    }
  }

  return ''
}

export function createPasswordPolicyValidator(getContext = () => ({}), getServerMessage = () => '') {
  return (rule, value, callback) => {
    const context = typeof getContext === 'function' ? getContext() : getContext
    const policyMessage = getPasswordPolicyMessage(value, context)
    const serverMessage = typeof getServerMessage === 'function'
      ? getServerMessage(value, context)
      : getServerMessage
    const message = policyMessage || serverMessage
    if (message) {
      callback(new Error(message))
      return
    }
    callback()
  }
}

export function validatePasswordPolicy(value, context = {}) {
  return getPasswordPolicyMessage(value, context)
}
