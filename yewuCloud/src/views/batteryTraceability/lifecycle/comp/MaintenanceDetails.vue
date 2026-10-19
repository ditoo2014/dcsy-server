<template>
  <div class="lifecycleRecordsCss">
    <div class="titleCss">
      <div class="titleTextCss">电池信息</div>
    </div>
    <div class="batteryMessageContentCss">
      <div class="batteryMessageCss">电池ID： {{ formData.batteryCode }}</div>
      <div class="batteryMessageCss">所属公司： {{ formData.province_company }}</div>
    </div>
    <div class="recordContentCss">
      <div class="titleContentCss">
        <div class="titleIconCss">
          <svg-icon style="color:#DFDFDF;width:1.2vw;height:1.2vw" icon-class="电池状态icon" />
        </div>
        <div class="recordTitleCss">电池当前状态</div>
      </div>
      <div class="currentStatusCss">
        <img v-if="currentBatteryStateImage" :src="currentBatteryStateImage" :alt="statusName">
        <span v-else>未知状态</span>
      </div>
    </div>
    <div class="recordContentCss">
      <div class="titleContentCss">
        <div class="titleIconCss">
          <svg-icon style="color:#DFDFDF;width:1.2vw;height:1.2vw" icon-class="流转记录icon" />
        </div>
        <div class="recordTitleCss">运维详情</div>
      </div>
      <div class="currentStatusRecordsCss">
        <div class="ywcontext">运维时间：{{ formData.maintenanceTime }}</div>
        <div class="ywcontext">运维描述：{{ formData.maintenanceDescription }}</div>
        <div class="ywcontext"><span>运维详情：</span><el-input
          :model-value="formData.faultDetail"
          style="width: 70%;font-size:1vw;font-weight:bold;"
          :disabled="true"
          type="textarea"
          :rows="4"
        />
        </div>
        <div class="ywcontext"><span>运维图片：</span><el-image
          style="width: 70%;"
          :src="formData.imgUrl"
          :preview-src-list="formData.imgUrls || (formData.imgUrl ? [formData.imgUrl] : [])"
        />
        </div>
      </div>
    </div>

  </div>
</template>

<script>
import inboundImage from '../imgs/lifecycle/stockIn.png'
import outboundImage from '../imgs/lifecycle/stockOut.png'
import onlineImage from '../imgs/lifecycle/siteCommissioning.png'
import offlineImage from '../imgs/lifecycle/siteDecommissioning.png'
import readyImage from '../imgs/lifecycle/inspectionReady.png'
import retiredImage from '../imgs/lifecycle/inspectionRetired.png'

// 保留原来的状态图片与布局，只按接口中的当前状态选择图片。
const lifecycleStates = {
  1: { name: '入库', image: inboundImage },
  2: { name: '出库', image: outboundImage },
  3: { name: '上站入网', image: onlineImage },
  4: { name: '退网下站', image: offlineImage },
  5: { name: '检测待用', image: readyImage },
  6: { name: '检测淘汰', image: retiredImage }
}

export default {
  props: {
    formData: {
      type: Object,
      default: () => {
        return {}
      }
    }
  },
  computed: {
    currentBatteryStateImage() {
      return lifecycleStates[Number(this.formData.positionStatus)]?.image || ''
    },
    statusName() {
      return lifecycleStates[Number(this.formData.positionStatus)]?.name || '未知状态'
    }
  }
}
</script>

<style lang="stylus" scoped>

.ywcontext {
    font-size 1vw
    font-weight bold
    padding 0.5vw 0 0.5vw 1.1vw
    display: flex;
}

.lifecycleRecordsCss {
  overflow auto
  width 100%
  .titleCss {
    width 100%
    height 3vw
    border-bottom 2px solid #e7e7e7
    .titleTextCss {
      margin-left 1vw
      padding-left 1vw
      border-left 2px solid #6902FF
      font-weight bold
      font-size 1.2vw
    }
  }
  .batteryMessageContentCss {
    padding 0.2vw 0 0.2vw 2vw
    border-bottom 1px solid #e7e7e7
    .batteryMessageCss {
      height 2vw
      line-height 2vw
      font-size 1vw
      font-weight bold
    }
  }
  .recordContentCss {
    padding-bottom 0.5vw
    width 100%
    // border-bottom 1px solid #e7e7e7
    .titleContentCss {
      padding-top 0.5vw
      width 100%
      height 3vw
      border-bottom 1px solid #e7e7e7
      .titleIconCss {
        float left
        margin 0.25vw 0.2vw 0 0.8vw
      }
      .recordTitleCss {
        float left
        font-size 1.2vw
        font-weight bold
      }
    }
    .currentStatusCss {
      margin-left 2vw
      width 30vw
      height 4.87vw
    }
    .currentStatusRecordsCss {
      padding-left 1vw
      height 20vw
      width 100%
    }
  }
}

img {
  width 100%
  height 100%
}
</style>
