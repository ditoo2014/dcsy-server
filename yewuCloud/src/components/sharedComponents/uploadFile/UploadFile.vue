<template>
  <div>
    <el-upload
      ref="upload"
      class="upload-demo"
      :multiple="false"
      action="system/sys/target/upload"
      :http-request="customUpload"
      :file-list="fileList"
      :on-remove="handleRemove"
      :limit="limit"
      :auto-upload="true"
    >
      <template #trigger>
        <el-button size="small" type="primary">{{ butName }}</el-button>
      </template>
    </el-upload>
  </div>
</template>

<script>
export default {
  props: {
    id: {
      type: [String, Number],
      default: ''
    },
    limit: { // 上传文件的最大个数
      type: Number,
      default: 10
    },
    butName: { // 上传文件按钮名称
      type: String,
      default: '选择文件'
    },
    fileNameList: {
      type: Array,
      default: () => {
        return [
          // { name: '' }
        ]
      }
    }
  },
  data() {
    return {
      fileList: [ // 组态文件列表
        // { name: '' }
      ]
    }
  },
  watch: {
    fileNameList: {
      handler(newValue, oldValue) {
        this.fileList = newValue
      },
      deep: true // 设为true时监听对象属性值变化
    }
  },
  methods: {
    customUpload(file) { // 选择文件
      this.fileList.push(file.file)
      this.$emit('selectFile', {
        id: this.id,
        file: file.file,
        fileList: this.fileList
      })
    },
    handleRemove(file, fileList) { // 删掉选择的文件
      this.fileList = fileList
      this.$emit('selectFile', {
        id: this.id,
        file: file,
        fileList: this.fileList
      })
    }
  }
}
</script>
