<template>
  <div>
    <RealtimeAlarms
      :equip-id="formData.id"
      @alarm-count-change="$emit('alarm-count-change', $event)"
    />
  </div>
</template>

<script>
import RealtimeAlarms from '../RealtimeAlarms.vue'

export default {
  components: {
    RealtimeAlarms
  },
  props: {
    formData: {
      type: Object,
      default: () => {}
    }
  },
  emits: ['alarm-count-change']
}
</script>
