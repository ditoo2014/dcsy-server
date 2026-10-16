<template>
  <div>
    <RealtimeWarnings
      :equip-id="formData.id"
      @warning-count-change="$emit('warning-count-change', $event)"
    />
  </div>
</template>

<script>
import RealtimeWarnings from '../RealtimeWarnings.vue'

export default {
  components: {
    RealtimeWarnings
  },
  emits: ['warning-count-change'],
  props: {
    formData: {
      type: Object,
      default: () => {}
    }
  }
}
</script>
