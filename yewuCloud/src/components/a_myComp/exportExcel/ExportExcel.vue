<template>
  <div />
</template>

<script>
export default {
  methods: {
    // 导出方法
    exportExcel(excelData) {
      const {
        multiHeader, // 标题
        header, // 表头
        filterVal, // 数据属性
        list, // 请求来的数据
        merges, // 需要合并的单元格
        filename // 文件名
      } = excelData
      import('./Export2Excel').then(excel => {
        const data = list.map(item => filterVal.map(j => item[j])) // 转换二维数组
        excel.export_json_to_excel({
          multiHeader, // 标题--非必要
          header, // 表头
          data, // 具体数据--二维数组
          merges, // 合并--非必要
          filename // 下载文件名
        })
      })
    },
    exportMultiSheet(excelData) { // 导出多个工作表sheet
      const userAoa = [
        ['用户数据', null, null], // 第 1 行大标题
        ['姓名', '年龄', '地址'], // 第 2 行真实列头
        ['张三', 25, '北京'],
        ['李四', 30, '上海']
      ]
      const userMerges = ['A1:C1'] // 合并第 1 行 A1~C1
      const prodAoa = [
        ['产品清单', null, null], // 大标题
        ['产品', '单价（元）', '库存'], // 列头
        ['苹果', 5, 100],
        ['香蕉', 3, 200]
      ]
      const prodMerges = ['A1:C1']
      const {
        aoaDataList = [userAoa, prodAoa], // 每份数据二维数组（含表头）
        sheetNames = ['用户数据', '产品数据'], // 与 sheetData 一一对应的 sheet 名
        mergesList = [userMerges, prodMerges], // 文件名，默认 multi-sheet
        filename = '复杂表头多Sheet' // 文件类型，默认 xlsx
      } = excelData

      import('./Export2Excel').then(excel => {
        excel.export_aoa_to_multi_sheet({
          aoaDataList,
          sheetNames,
          mergesList,
          filename
        })
      })
    }
  }
}
</script>
