<template>
  <div class="tagsViewContainerCss">
    <div
      v-for="tag in visitedViews"
      :key="tag.path"
      class="tagsViewItemCss"
      :class="isActive(tag)?'activeCss':''"
      @click="selectedNowTag(tag)"
      @contextmenu.prevent="openMenu(tag, $event)"
    >
      <span>{{ tag.title }}</span>
      <div v-if="!tag.meta.affix && visitedViews.length > 1" class="closeIconContentCss" @click.prevent.stop="closeSelectedTag(tag)">
        <img :src="isActive(tag) ? closePrimaryImage : closeImage">
      </div>
    </div>
    <ul v-show="visible" :style="{top,left}" class="contextmenuCss">
      <li v-if="!(selectedTag.meta&&selectedTag.meta.affix)" @click="closeSelectedTag(selectedTag)">关闭</li>
      <li @click="closeOthersTags">关闭其他</li>
      <li @click="closeAllTags(selectedTag)">关闭全部</li>
    </ul>
  </div>
</template>

<script>
import { resolveRoutePath } from '@/utils/resolve-route-path'
import closeImage from './close.png'
import closePrimaryImage from './closePrimary.png'

export default {
  data() {
    return {
      closeImage,
      closePrimaryImage,
      visible: false,
      top: 0,
      left: 0,
      selectedTag: {},
      affixTags: []
    }
  },
  computed: {
    visitedViews() {
      return this.$store.state.tagsView.visitedViews
    },
    routes() {
      return this.$store.state.permission.routes
    }
  },
  watch: {
    $route() {
      this.addTags()
    },
    visible(value) {
      if (value) {
        document.body.addEventListener('click', this.closeMenu)
      } else {
        document.body.removeEventListener('click', this.closeMenu)
      }
    }
  },
  mounted() {
    this.initTags()
    this.addTags()
  },
  methods: {
    isActive(route) {
      return route.path === this.$route.path
    },
    filterAffixTags(routes, basePath = '/') {
      let tags = []
      routes.forEach(route => {
        if (route.meta && route.meta.affix) {
          const tagPath = resolveRoutePath(basePath, route.path)
          tags.push({
            fullPath: tagPath,
            path: tagPath,
            name: route.name,
            meta: { ...route.meta }
          })
        }
        if (route.children) {
          const tempTags = this.filterAffixTags(route.children, route.path)
          if (tempTags.length >= 1) {
            tags = [...tags, ...tempTags]
          }
        }
      })
      return tags
    },
    initTags() { // 初始化 affixTags
      const affixTags = this.affixTags = this.filterAffixTags(this.routes)
      for (const tag of affixTags) {
        // Must have tag name
        if (tag.name) {
          this.$store.dispatch('tagsView/addVisitedView', tag)
        }
      }
    },
    addTags() {
      const { name } = this.$route
      if (name) {
        this.$store.dispatch('tagsView/addView', this.$route)
      }
      return false
    },
    closeSelectedTag(view) { // 关闭标签
      this.$store.dispatch('tagsView/delView', view).then(({ visitedViews }) => {
        if (this.isActive(view)) {
          this.toLastView(visitedViews, view)
        }
      })
    },
    closeOthersTags() { // 关闭其他标签
      this.$router.push(this.selectedTag)
      this.$store.dispatch('tagsView/delOthersViews', this.selectedTag).then(() => {
      })
    },
    closeAllTags(view) { // 关闭所有标签
      this.$store.dispatch('tagsView/delAllViews').then(({ visitedViews }) => {
        if (this.affixTags.some(tag => tag.path === view.path)) {
          return
        }
        this.toLastView(visitedViews, view)
      })
    },
    toLastView(visitedViews, view) {
      const latestView = visitedViews.slice(-1)[0]
      if (latestView) {
        this.$router.push(latestView)
      } else {
        this.$router.push('/')
      }
    },
    selectedNowTag(tag) { // 点击当前标签
      this.$router.push({ path: tag.path, query: tag.query })
    },
    openMenu(tag, e) { // 右键点击事件
      const menuWidth = this.$el.getBoundingClientRect().left // 左侧导航条宽度
      this.top = e.clientY / 1920 + 5 + 'vw' // 1920为屏幕分辨率
      this.left = e.clientX - menuWidth + 'px'
      this.visible = true
      this.selectedTag = tag
    },
    closeMenu() {
      this.visible = false
    }
  }
}
</script>

<style lang="stylus" scoped>
tagsViewContainerHeight = 2.239583vw
.tagsViewContainerCss {
  width 100%
  height tagsViewContainerHeight
  .tagsViewItemCss {
    float left
    padding 0 0.9375vw
    width 8.4375vw
    height tagsViewContainerHeight
    border 1px solid #EAEDF7
    border-radius 0.52vw 0.52vw 0 0
    cursor pointer
    color #636E95
    &.activeCss {
      background-color #369AFE
      color #fff
    }
    span {
      font-size 0.72916vw
      font-weight 500
      line-height tagsViewContainerHeight
    }
    .closeIconContentCss {
      float right
      margin-top 0.677vw
      width 0.9375vw
      height 0.9375vw
      line-height 0
      img {
        width 100%
        height 100%
      }
    }
  }
  .contextmenuCss {
    background-color #fff
    z-index 3000
    position absolute
    list-style-type none
    padding 5px 0
    border-radius 4px
    font-size 12px
    font-weight 400
    color #333
    box-shadow 2px 2px 3px 0 rgba(0, 0, 0, .3)
    li {
      margin 0
      padding 7px 16px
      cursor pointer
      &:hover {
        background: #eee
      }
    }
  }
}
</style>
