<template>
  <div>
    <Yjxxssyj
      :equip-id="formData.id"
      @warning-count-change="$emit('warning-count-change', $event)"
    />
  </div>
</template>

<script>
import Yjxxssyj from '../yjxxssyj.vue'

export default {
  components: {
    Yjxxssyj
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
