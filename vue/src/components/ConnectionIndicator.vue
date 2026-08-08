<script setup lang="ts">
/**
 * Connection indicator.
 *
 * "A dispatch desk that looks live but is frozen is worse than one that admits
 * it" — PRD 004. Unobtrusive when healthy, clearly degraded when not. It renders
 * state and nothing else: reconnection policy belongs to the transport.
 */
import { useConnectionState } from '@/composables/useConnectionState'

const { state, label, description, isDisconnected } = useConnectionState()
</script>

<template>
  <span
    class="live-indicator"
    :class="[`live-indicator--${state}`, { 'live-indicator--alert': isDisconnected }]"
    :title="description"
    role="status"
    :aria-label="`Forbindelse: ${label}. ${description}`"
  >
    <span class="live-indicator__dot" aria-hidden="true" />
    <span class="live-indicator__label">{{ label }}</span>
  </span>
</template>

<style scoped>
.live-indicator {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  line-height: 1;
  white-space: nowrap;
  opacity: 0.75;
}

.live-indicator__dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 9999px;
  background-color: currentColor;
  flex: none;
}

.live-indicator--live {
  color: #16a34a;
}

.live-indicator--polling {
  color: #a2aeb2;
}

.live-indicator--reconnecting {
  color: #d97706;
}

.live-indicator--offline {
  color: #dc2626;
}

/* Losing updates is the one case worth pulling the eye. */
.live-indicator--alert {
  opacity: 1;
  font-weight: 600;
}

.live-indicator--reconnecting .live-indicator__dot,
.live-indicator--offline .live-indicator__dot {
  animation: live-indicator-pulse 1.4s ease-in-out infinite;
}

@keyframes live-indicator-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.25;
  }
}

/* Respect a reduced-motion preference: the colour already carries the meaning. */
@media (prefers-reduced-motion: reduce) {
  .live-indicator__dot {
    animation: none;
  }
}

/* The label is redundant on narrow screens; the dot and tooltip carry it. */
@media (max-width: 640px) {
  .live-indicator__label {
    display: none;
  }
}
</style>
