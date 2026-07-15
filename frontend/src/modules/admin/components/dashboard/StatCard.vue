<script setup lang="ts">
import { computed, type Component } from 'vue'
import { ArrowDownRight, ArrowUpRight, Minus } from 'lucide-vue-next'
import type { DashboardColorToken } from '../../types/dashboard'
import { DELTA_TEXT_CLASSES, METRIC_ICON_CLASSES, type DeltaDirection } from '../../utils/dashboard'

const props = defineProps<{
  label: string
  value: string
  icon: Component
  color: DashboardColorToken
  deltaDirection: DeltaDirection
  deltaText: string
  deltaCaption: string
  clickable?: boolean
}>()

const emit = defineEmits<{
  (event: 'click'): void
}>()

const iconClass = computed(() => METRIC_ICON_CLASSES[props.color])
const deltaClass = computed(() => DELTA_TEXT_CLASSES[props.deltaDirection])
const deltaIcon = computed(() => {
  if (props.deltaDirection === 'up') return ArrowUpRight
  if (props.deltaDirection === 'down') return ArrowDownRight
  return Minus
})
</script>

<template>
  <component
    :is="clickable ? 'button' : 'div'"
    :type="clickable ? 'button' : undefined"
    class="w-full rounded-xl border border-border/50 bg-card p-5 text-left shadow-sm"
    :class="{ 'cursor-pointer transition-[border-color,box-shadow,transform] hover:border-primary/30 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background active:translate-y-px': clickable }"
    @click="clickable && emit('click')"
  >
    <div class="flex items-start justify-between">
      <div class="min-w-0">
        <p class="text-sm font-medium text-muted-foreground">{{ label }}</p>
        <p class="mt-2 break-words text-2xl font-bold leading-tight tabular-nums text-foreground 2xl:text-3xl">{{ value }}</p>
      </div>
      <div :class="['p-3 rounded-xl shrink-0', iconClass]">
        <component :is="icon" class="w-6 h-6" aria-hidden="true" />
      </div>
    </div>
    <div class="mt-4 flex items-center gap-1.5 text-xs">
      <span v-if="deltaText" :class="['inline-flex items-center gap-0.5 font-semibold', deltaClass]">
        <component :is="deltaIcon" class="w-3.5 h-3.5" />
        {{ deltaText }}
      </span>
      <span class="text-muted-foreground">{{ deltaCaption }}</span>
    </div>
  </component>
</template>
