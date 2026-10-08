<template>
  <!-- 必须用 ElTree 显式引用，避免 <el-tree> 被解析为本地 MyElTree 导致栈溢出 -->
  <ElTree
    ref="treeRef"
    :data="treeData"
    node-key="id"
    :empty-text="emptyText"
    :show-checkbox="showCheckbox"
    :props="treeProps"
    :current-node-key="currentNodeKey"
    highlight-current
    :expand-on-click-node="false"
    class="el-nav-tree"
    @node-click="handleNodeClick"
    @check="onCheckChange"
  >
    <template #default="{ data }">
      <span class="tree-node">
        <el-icon v-if="data.titleIconType" class="tree-node-icon">
          <component :is="iconComponent(data.titleIconType)" />
        </el-icon>
        <span :style="{ color: data.titleColor || '' }">{{ data.title }}</span>
      </span>
    </template>
  </ElTree>
</template>

<script>
import { ElTree } from 'element-plus'
import { Folder, Location } from '@element-plus/icons-vue'

const textColor = '#1890FF'

const ICON_MAP = {
  'md-albums': Folder,
  'ios-locate': Location
}

export default {
  name: 'MyElTree',
  components: { ElTree },
  props: {
    emptyText: {
      type: String,
      default: '请创建节点'
    },
    naviNameList: {
      type: Array,
      default: () => [
        { level: 0, parentId: 0, id: 1, name: '企业' }
      ]
    },
    showCheckbox: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      treeData: [],
      activeExpandTargetId: null,
      treeProps: {
        label: 'title',
        children: 'children'
      }
    }
  },
  computed: {
    currentNodeKey() {
      if (this.activeExpandTargetId != null) {
        return this.activeExpandTargetId
      }
      const selected = this.naviNameList.find(item => item.selected === true)
      return selected ? selected.id : null
    }
  },
  watch: {
    naviNameList() {
      const selected = this.naviNameList.find(item => item.selected === true)
      this.activeExpandTargetId = selected ? selected.id : null
      this.createdTree()
    }
  },
  created() {
    const selected = this.naviNameList.find(item => item.selected === true)
    this.activeExpandTargetId = selected ? selected.id : null
    this.createdTree()
  },
  methods: {
    /** 获取选中节点及其祖先 id（整条路径均需展开才能看到选中项） */
    getExpandedKeys(targetId) {
      let selectedItem
      if (targetId != null && targetId !== '') {
        selectedItem = this.naviNameList.find(item => String(item.id) === String(targetId))
      }
      if (!selectedItem) {
        selectedItem = this.naviNameList.find(item => item.selected === true)
      }
      if (!selectedItem) return []

      const keys = [selectedItem.id]
      let parentId = selectedItem.parentId
      while (parentId != null && parentId !== 0 && parentId !== '') {
        keys.push(parentId)
        const parent = this.naviNameList.find(item => String(item.id) === String(parentId))
        if (!parent) break
        parentId = parent.parentId
      }
      return keys
    },
    /** 通过 store 控制展开，仅展开选中节点及其祖先路径 */
    applyExpandedState() {
      this.$nextTick(() => {
        this.$nextTick(() => {
          const tree = this.$refs.treeRef
          if (!tree?.store?.nodesMap) return

          const keysToExpand = this.getExpandedKeys(this.activeExpandTargetId)

          Object.values(tree.store.nodesMap).forEach(node => {
            if (node) node.expanded = false
          })
          keysToExpand.forEach(key => {
            let node = tree.getNode(key)
            if (!node && key != null) node = tree.getNode(String(key))
            if (!node && key != null) node = tree.getNode(Number(key))
            if (node) node.expanded = true
          })

          const currentKey = this.currentNodeKey
          if (currentKey != null) {
            tree.setCurrentKey(currentKey)
          }
        })
      })
    },
    iconComponent(type) {
      return ICON_MAP[type] || Location
    },
    createdTree() {
      const treeData = []
      this.naviNameList.forEach(item => {
        const level = parseInt(item.level)
        const titleColor = item.selected ? textColor : (item.titleColor || '')
        const expand = false
        switch (level) {
          case 0:
            treeData.push(this.treeNodeData({
              level: item.level, parentId: item.parentId, id: item.id, expand,
              title: item.name, titleColor, selected: item.selected, titleIconType: 'md-albums'
            }))
            break
          case 1:
            treeData.forEach(parent => {
              if (parent.id === item.parentId) {
                parent.children.push(this.treeNodeData({
                  level: item.level, parentId: item.parentId, id: item.id, expand,
                  title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                }))
              }
            })
            break
          case 2:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parent => {
                if (parent.id === item.parentId) {
                  parent.children.push(this.treeNodeData({
                    level: item.level, parentId: item.parentId, id: item.id, expand,
                    title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                  }))
                }
              })
            })
            break
          case 3:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parent => {
                  if (parent.id === item.parentId) {
                    parent.children.push(this.treeNodeData({
                      level: item.level, parentId: item.parentId, id: item.id, expand,
                      title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                    }))
                  }
                })
              })
            })
            break
          case 4:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parentThree => {
                  parentThree.children.forEach(parent => {
                    if (parent.id === item.parentId) {
                      parent.children.push(this.treeNodeData({
                        level: item.level, parentId: item.parentId, id: item.id, expand,
                        title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                      }))
                    }
                  })
                })
              })
            })
            break
          case 5:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parentThree => {
                  parentThree.children.forEach(parentFour => {
                    parentFour.children.forEach(parent => {
                      if (parent.id === item.parentId) {
                        parent.children.push(this.treeNodeData({
                          level: item.level, parentId: item.parentId, id: item.id, expand,
                          title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                        }))
                      }
                    })
                  })
                })
              })
            })
            break
          case 6:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parentThree => {
                  parentThree.children.forEach(parentFour => {
                    parentFour.children.forEach(parentFive => {
                      parentFive.children.forEach(parent => {
                        if (parent.id === item.parentId) {
                          parent.children.push(this.treeNodeData({
                            level: item.level, parentId: item.parentId, id: item.id, expand,
                            title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                          }))
                        }
                      })
                    })
                  })
                })
              })
            })
            break
          case 7:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parentThree => {
                  parentThree.children.forEach(parentFour => {
                    parentFour.children.forEach(parentFive => {
                      parentFive.children.forEach(parentSix => {
                        parentSix.children.forEach(parent => {
                          if (parent.id === item.parentId) {
                            parent.children.push(this.treeNodeData({
                              level: item.level, parentId: item.parentId, id: item.id, expand,
                              title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                            }))
                          }
                        })
                      })
                    })
                  })
                })
              })
            })
            break
          case 8:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parentThree => {
                  parentThree.children.forEach(parentFour => {
                    parentFour.children.forEach(parentFive => {
                      parentFive.children.forEach(parentSix => {
                        parentSix.children.forEach(parentSeven => {
                          parentSeven.children.forEach(parent => {
                            if (parent.id === item.parentId) {
                              parent.children.push(this.treeNodeData({
                                level: item.level, parentId: item.parentId, id: item.id, expand,
                                title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                              }))
                            }
                          })
                        })
                      })
                    })
                  })
                })
              })
            })
            break
          case 9:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parentThree => {
                  parentThree.children.forEach(parentFour => {
                    parentFour.children.forEach(parentFive => {
                      parentFive.children.forEach(parentSix => {
                        parentSix.children.forEach(parentSeven => {
                          parentSeven.children.forEach(parentEight => {
                            parentEight.children.forEach(parent => {
                              if (parent.id === item.parentId) {
                                parent.children.push(this.treeNodeData({
                                  level: item.level, parentId: item.parentId, id: item.id, expand,
                                  title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                                }))
                              }
                            })
                          })
                        })
                      })
                    })
                  })
                })
              })
            })
            break
          case 10:
            treeData.forEach(parentOne => {
              parentOne.children.forEach(parentTwo => {
                parentTwo.children.forEach(parentThree => {
                  parentThree.children.forEach(parentFour => {
                    parentFour.children.forEach(parentFive => {
                      parentFive.children.forEach(parentSix => {
                        parentSix.children.forEach(parentSeven => {
                          parentSeven.children.forEach(parentEight => {
                            parentEight.children.forEach(parentNine => {
                              parentNine.children.forEach(parent => {
                                if (parent.id === item.parentId) {
                                  parent.children.push(this.treeNodeData({
                                    level: item.level, parentId: item.parentId, id: item.id, expand,
                                    title: item.name, titleColor, selected: item.selected, titleIconType: 'ios-locate'
                                  }))
                                }
                              })
                            })
                          })
                        })
                      })
                    })
                  })
                })
              })
            })
            break
        }
      })
      this.treeData = treeData
      this.applyExpandedState()
    },
    look(data) {
      const clearTitleColor = (nodes) => {
        if (!nodes) return
        nodes.forEach(node => {
          node.titleColor = ''
          clearTitleColor(node.children)
        })
      }
      clearTitleColor(this.treeData)

      if (data.titleColor === '') {
        data.titleColor = textColor
        this.activeExpandTargetId = data.id
      } else {
        data.titleColor = ''
        this.activeExpandTargetId = null
      }

      this.applyExpandedState()
      this.$emit('clickTree', data)
    },
    handleNodeClick(data) {
      if (data) {
        this.look(data)
      }
    },
    treeNodeData(params) {
      const {
        level, parentId, id, title, titleColor = '', titleIconType = '',
        selected = false, expand = true
      } = params
      return {
        level,
        parentId,
        id,
        title,
        titleColor,
        titleIconType,
        expand,
        selected,
        children: []
      }
    },
    onCheckChange(data, checkedInfo) {
      this.$emit('onCheckChange', checkedInfo.checkedNodes || data)
    }
  }
}
</script>

<style lang="stylus" scoped>
.el-nav-tree
  background transparent

.tree-node
  display inline-flex
  align-items center

.tree-node-icon
  margin-right 5px
  vertical-align middle
</style>
