/**
 * ECharts 按需引入（业务侧统一使用 this.$echarts 或 import echarts from '@/utils/echarts'）
 */
import * as echarts from 'echarts/core'
import { BarChart, LineChart, MapChart, PieChart } from 'echarts/charts'
import {
  DataZoomComponent,
  GeoComponent,
  GraphicComponent,
  GridComponent,
  LegendComponent,
  MarkLineComponent,
  TitleComponent,
  TooltipComponent,
  VisualMapComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { LabelLayout } from 'echarts/features'

echarts.use([
  BarChart,
  LineChart,
  PieChart,
  MapChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  VisualMapComponent,
  GeoComponent,
  MarkLineComponent,
  GraphicComponent,
  // 按需构建需要注册标签布局，否则柱状图 hideOverlap 不会生效。
  LabelLayout,
  CanvasRenderer
])

export default echarts
