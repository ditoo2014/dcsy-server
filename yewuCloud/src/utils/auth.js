import Cookies from 'js-cookie'
import { parseUserInfo as parseUserInfoFromStorage } from '@/utils/safeJson.js'

const TokenKey = 'access_token'
export function getToken() {
  return Cookies.get(TokenKey)
}
export function setToken(token) {
  return Cookies.set(TokenKey, token)
}
export function removeToken() {
  return Cookies.remove(TokenKey)
}

const UsernameKey = 'username'
export function getUsername() {
  return localStorage.getItem(UsernameKey)
}
export function setUsername(username) {
  return localStorage.setItem(UsernameKey, username)
}
export function removeUsername() {
  return localStorage.removeItem(UsernameKey)
}

const userInfoStrKey = 'userInfoStr'
export function getUserInfoStr() {
  return localStorage.getItem(userInfoStrKey)
}
export function setUserInfoStr(userInfoStr) {
  return localStorage.setItem(userInfoStrKey, userInfoStr)
}
export function removeUserInfoStr() {
  return localStorage.removeItem(userInfoStrKey)
}

const userIdKey = 'userId'
export function getUserId() {
  const stored = localStorage.getItem(userIdKey)
  if (stored) {
    return stored
  }
  const user = parseUserInfoFromStorage(null)
  return user && user.id != null ? String(user.id) : ''
}
export function setUserId(userId) {
  if (userId == null || userId === '') {
    localStorage.removeItem(userIdKey)
    return
  }
  localStorage.setItem(userIdKey, String(userId))
}
export function removeUserId() {
  localStorage.removeItem(userIdKey)
}

const roleIdKey = 'roleId'
export function getRoleId() {
  const stored = localStorage.getItem(roleIdKey)
  if (stored) {
    return stored
  }
  const user = parseUserInfoFromStorage(null)
  if (user && user.roleId != null) {
    return String(user.roleId)
  }
  return null
}
export function setRoleId(roleId) {
  if (roleId == null || roleId === '') {
    localStorage.removeItem(roleIdKey)
    return
  }
  localStorage.setItem(roleIdKey, String(roleId))
}
export function removeRoleId() {
  localStorage.removeItem(roleIdKey)
}

const mydbKey = 'mydb'
export function setMydb(mydb) {
  if (mydb == null || mydb === '') {
    localStorage.removeItem(mydbKey)
    return
  }
  localStorage.setItem(mydbKey, String(mydb))
}
export function getMydb() {
  return localStorage.getItem(mydbKey)
}
export function removeMydb() {
  localStorage.removeItem(mydbKey)
}

const collectionTokenStorageKey = 'cjyToken'
export function getCollectionToken() {
  return localStorage.getItem(collectionTokenStorageKey) || ''
}
export function setCollectionToken(token) {
  if (!token) {
    localStorage.removeItem(collectionTokenStorageKey)
    return
  }
  localStorage.setItem(collectionTokenStorageKey, token)
}
export function removeCollectionToken() {
  localStorage.removeItem(collectionTokenStorageKey)
}

const pagelistKey = 'pagelist'

export function getPageList() {
  return localStorage.getItem(pagelistKey)
}

export function setPageList(pageList) {
  return localStorage.setItem(pagelistKey, pageList)
}

export function removePageList() {
  return localStorage.removeItem(pagelistKey)
}
