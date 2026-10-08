<template>
  <div>
    <Gjxxssgj
      :equip-id="formData.id"
      @alarm-count-change="$emit('alarm-count-change', $event)"
    />
  </div>
</template>

<script>
import Gjxxssgj from '../gjxxssgj.vue'

export default {
  components: {
    Gjxxssgj
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
