function corpKey(value) {
  return value == null || value === '' ? '' : String(value)
}

function boundCorpKeys(userInfo) {
  const corpIds = Array.isArray(userInfo && userInfo.corpIds) ? userInfo.corpIds : []
  return corpIds.map(corpKey).filter(Boolean)
}

export function buildQueryableCorpKeys(treeList = [], userInfo = {}) {
  if (Number(userInfo.isadmin) === 1) {
    return treeList.map(item => corpKey(item && item.id)).filter(Boolean)
  }

  const queryableKeys = new Set(boundCorpKeys(userInfo))
  let changed = true
  while (changed) {
    changed = false
    treeList.forEach(item => {
      const idKey = corpKey(item && item.id)
      const parentKey = corpKey(item && item.parentId)
      if (idKey && parentKey && queryableKeys.has(parentKey) && !queryableKeys.has(idKey)) {
        queryableKeys.add(idKey)
        changed = true
      }
    })
  }
  return Array.from(queryableKeys)
}

export function findDefaultCorp(treeList = [], userInfo = {}, queryableCorpKeys = []) {
  const itemMap = new Map()
  treeList.forEach(item => {
    const key = corpKey(item && item.id)
    if (key) itemMap.set(key, item)
  })

  const boundCorp = boundCorpKeys(userInfo)
    .map(key => itemMap.get(key))
    .find(Boolean)
  if (boundCorp) return boundCorp

  const queryableSet = new Set(queryableCorpKeys)
  const queryableCorp = treeList.find(item => queryableSet.has(corpKey(item && item.id)))
  if (queryableCorp) return queryableCorp

  return Number(userInfo.isadmin) === 1 ? treeList.find(item => corpKey(item && item.id)) : null
}

export function resolveCorpQueryId(selectedCorpId, queryableCorpKeys = [], userInfo = {}) {
  const selectedKey = corpKey(selectedCorpId)
  if (!selectedKey) return undefined
  if (Number(userInfo.isadmin) === 1) return selectedCorpId
  return queryableCorpKeys.includes(selectedKey) ? selectedCorpId : undefined
}

export function expandCorpKeys(treeList = [], corpId) {
  const rootKey = corpKey(corpId)
  if (!rootKey) return []

  const scope = new Set([rootKey])
  let changed = true
  while (changed) {
    changed = false
    treeList.forEach(item => {
      const idKey = corpKey(item && item.id)
      const parentKey = corpKey(item && item.parentId)
      if (idKey && parentKey && scope.has(parentKey) && !scope.has(idKey)) {
        scope.add(idKey)
        changed = true
      }
    })
  }
  return Array.from(scope)
}
