<template>
  <div>
    <div class="hamburgerAndbreadcrumbCss">
      <div class="hamburgerContentCss">
        <Hamburger :is-active="sidebar.opened" @toggle-click="toggleSideBar" />
      </div>
      <div class="breadcrumbContainerCss">
        <Breadcrumb />
      </div>
    </div>
    <div class="TagsViewContentCss">
      <TagsView />
    </div>
  </div>
</template>

<script>
import { mapGetters } from '@/store/pinia-helpers'
import Hamburger from '@/components/Hamburger'
import Breadcrumb from '@/components/Breadcrumb'
import TagsView from './TagsView'

export default {
  components: {
    Hamburger,
    Breadcrumb,
    TagsView
  },
  computed: {
    ...mapGetters([
      'sidebar'
    ])
  },
  methods: {
    toggleSideBar() {
      this.$store.dispatch('app/toggleSideBar')
    }
  }
}
</script>

<style lang="stylus" scoped>
navbarHeight = 2.5vw

.hamburgerAndbreadcrumbCss {
  display flow-root
  width 100%
  height navbarHeight
  overflow hidden
  border-bottom 1px solid #EAEDF7
  background-color #fff
  .hamburgerContentCss {
    float left
    width 3%
    height navbarHeight
    line-height navbarHeight
    text-align center
  }
  .breadcrumbContainerCss {
    float left
  }
}
.TagsViewContentCss {
  display flow-root
  padding-top 0.468717vw
  width 100%
  border-bottom 1px solid #EAEDF7
  background-color #fff
}

// 窄屏保持导航可点击、面包屑可读，宽屏仍沿用原版比例。
@media (max-width: 1280px) {
  .hamburgerAndbreadcrumbCss {
    display flex
    align-items center
    height 40px
    .hamburgerContentCss {
      float none
      flex 0 0 40px
      width 40px
      height 40px
      line-height 40px
    }
    .breadcrumbContainerCss {
      float none
      min-width 0
      overflow-x auto
      :deep(.appBreadcrumbCss) {
        font-size 13px
        line-height 40px
        white-space nowrap
      }
    }
  }
}
</style>
