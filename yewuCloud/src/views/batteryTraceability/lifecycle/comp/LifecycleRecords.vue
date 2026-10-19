<template>
  <div class="lifecycleRecordsCss">
    <div class="titleCss">
      <div class="titleTextCss">电池信息</div>
    </div>
    <div class="batteryMessageContentCss">
      <div class="batteryMessageCss">电池ID： {{ formData.equipmentCode }}</div>
      <div class="batteryMessageCss">所属公司： {{ city_company }}</div>
    </div>
    <div class="recordContentCss">
      <div class="titleContentCss">
        <div class="titleIconCss">
          <svg-icon style="color:#DFDFDF;width:1.2vw;height:1.2vw" icon-class="电池状态icon" />
        </div>
        <div class="recordTitleCss">电池当前状态</div>
      </div>
      <div class="currentStatusCss">
        <img v-if="currentBatteryStateImage" :src="currentBatteryStateImage">
      </div>
    </div>
    <div class="recordContentCss">
      <div class="titleContentCss">
        <div class="titleIconCss">
          <svg-icon style="color:#DFDFDF;width:1.2vw;height:1.2vw" icon-class="流转记录icon" />
        </div>
        <div class="recordTitleCss">流转记录</div>
      </div>
      <div class="currentStatusRecordsCss">
        <el-steps direction="vertical" :active="0" process-status="success">
          <el-step
            v-for="(item, index) in circulationRecords"
            :key="index"
            :title="item.operateDetails.desc"
            :icon="$legacyIcon('el-icon-remove-outline')"
            :status="item.operateType === operateType ? '' : 'wait'"
          >
            <template #description>
              <span
                v-for="(msgItem, msgIndex) in item.msgList"
                :key="msgIndex"
              >
                <span>{{ msgItem }}</span>
                <br>
              </span>
            </template>
          </el-step>
        </el-steps>
      </div>
    </div>

  </div>
</template>

<script>
import { equipmentcirlog } from '@/api/monitoring.js'
import { strSetArr } from '@/utils/utils.js'
import inboundImage from '../imgs/lifecycle/stockIn.png'
import outboundImage from '../imgs/lifecycle/stockOut.png'
import onlineImage from '../imgs/lifecycle/siteCommissioning.png'
import offlineImage from '../imgs/lifecycle/siteDecommissioning.png'
import readyImage from '../imgs/lifecycle/inspectionReady.png'
import retiredImage from '../imgs/lifecycle/inspectionRetired.png'

const lifecycleImageMap = {
  1: inboundImage,
  2: outboundImage,
  3: onlineImage,
  4: offlineImage,
  5: readyImage,
  6: retiredImage
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
  data() {
    return {
      operateType: 0,
      currentBatteryStateImage: '',
      circulationRecords: [],
      city_company: '' // 省公司
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    async init() {
      if (!this.formData.id) {
        return
      }
      const res = await equipmentcirlog({
        equipmentId: this.formData.id
      })
      const list = res.return.data
      let operateType = 0 // 电子状态进度
      const circulationRecords = []
      if (list) {
        list.forEach(item => {
          // if (item.operateType > operateType) {
          operateType = item.operateType
          // }
          item.msgList = strSetArr(item.operateDetails.title, '\n')

          circulationRecords.push(item)
        })
      }
      console.log(circulationRecords)
      this.currentBatteryStateImage = lifecycleImageMap[operateType] || ''
      this.operateType = operateType

      // this.circulationRecords = objectSort(circulationRecords, 'operateType', false)
      this.circulationRecords = circulationRecords.reverse() // 颠倒
      this.city_company = circulationRecords[circulationRecords.length - 1].operateDetails.city_company
    }
  }
}
</script>

<style lang="stylus" scoped>
.lifecycleRecordsCss {
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
    border-bottom 1px solid #e7e7e7
    .titleContentCss {
      padding-top 0.5vw
      width 100%
      height 3vw
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
      overflow auto
      width 100%
    }
  }
}

img {
  width 100%
  height 100%
}
</style>
