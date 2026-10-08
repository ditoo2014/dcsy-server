<template>
  <div class="me-tree-transfer">
    <tree-transfer
      v-model:from-data="fromData"
      v-model:to-data="toData"
      :title-list="title"
      :default-props="defaultProps"
      root-pid="0"
      :default-expand-all="true"
      @add="add"
      @remove="remove"
    />
  </div>
</template>

<script>
import TreeTransfer from 'tree-transfer-vue3'

export default {
  components: {
    treeTransfer: TreeTransfer
  },
  props: {
    type: {
      type: [Number, String],
      default: ''
    },
    title: {
      type: Array,
      default: () => {
        return ['', '']
      }
    },
    leftData: {
      type: Array,
      default: () => {
        return []
      }
    },
    rightData: {
      type: Array,
      default: () => {
        return []
      }
    }
  },
  data() {
    return {
      defaultProps: {
        id: 'id',
        parentId: 'pid',
        label: 'label',
        children: 'children'
      },
      fromData: [],
      toData: []
    }
  },
  watch: {
    leftData: {
      handler(val) {
        this.fromData = this.getTreeData(val || [])
      },
      deep: true
    },
    rightData: {
      handler(val) {
        this.toData = this.getTreeData(val || [])
      },
      deep: true
    }
  },
  created() {
    this.fromData = this.getTreeData(this.leftData)
    this.toData = this.getTreeData(this.rightData)
  },
  methods: {
    add(fromData, toData) {
      this.fromData = fromData
      this.toData = toData
      this.$emit('selectData', {
        type: this.type,
        fromData,
        toData
      })
    },
    remove(fromData, toData) {
      this.fromData = fromData
      this.toData = toData
      this.$emit('selectData', {
        type: this.type,
        fromData,
        toData
      })
    },
    getTreeData(dataList) {
      const treeList = []
      dataList.forEach(item => {
        if (item.pid) {
          treeList.forEach(tree => {
            if (tree.id === item.pid) {
              tree.children.push({
                id: item.id,
                pid: item.pid,
                label: item.label,
                level: item.level,
                type: item.type,
                children: []
              })
            } else {
              tree.children.forEach(treeChild1 => {
                if (treeChild1.id === item.pid) {
                  treeChild1.children.push({
                    id: item.id,
                    pid: item.pid,
                    label: item.label,
                    level: item.level,
                    type: item.type,
                    children: []
                  })
                } else {
                  treeChild1.children.forEach(treeChild2 => {
                    if (treeChild2.id === item.pid) {
                      treeChild2.children.push({
                        id: item.id,
                        pid: item.pid,
                        label: item.label,
                        level: item.level,
                        type: item.type,
                        children: []
                      })
                    } else {
                      treeChild2.children.forEach(treeChild3 => {
                        if (treeChild3.id === item.pid) {
                          treeChild3.children.push({
                            id: item.id,
                            pid: item.pid,
                            label: item.label,
                            level: item.level,
                            type: item.type,
                            children: []
                          })
                        }
                      })
                    }
                  })
                }
              })
            }
          })
        } else {
          treeList.push({
            id: item.id,
            pid: item.pid || 0,
            label: item.label,
            level: item.level,
            type: item.type,
            children: []
          })
        }
      })
      return treeList
    }
  }
}

</script>

<style scoped>
.me-tree-transfer {
  width: 100%;
}
</style>
