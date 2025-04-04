<template>
  <div class="notification-container">
    <transition-group name="notification">
      <div
          v-for="notification in visibleNotifications"
          :key="notification.id"
          class="notification"
          :class="notification.type"
      >
        <div class="notification-content">
          <div class="notification-message">{{ notification.message }}</div>
          <div class="notification-time">{{ formatTime(notification.timestamp) }}</div>
        </div>
        <button class="notification-close" @click="dismissNotification(notification.id)">×</button>
      </div>
    </transition-group>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { notifications } from '../services/websocket';

// State for visible notifications
const visibleNotifications = ref([]);
const maxVisibleNotifications = 5;

// Watch for new notifications
onMounted(() => {
  // Initialize with current notifications
  updateVisibleNotifications();

  // Set up interval to check for new notifications
  setInterval(updateVisibleNotifications, 1000);
});

// Format time for display
function formatTime(date) {
  if (!(date instanceof Date)) {
    date = new Date(date);
  }

  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

// Update visible notifications
function updateVisibleNotifications() {
  visibleNotifications.value = notifications.slice(0, maxVisibleNotifications);
}

// Dismiss a notification
function dismissNotification(id) {
  const index = notifications.findIndex(n => n.id === id);
  if (index !== -1) {
    notifications.splice(index, 1);
    updateVisibleNotifications();
  }
}
</script>

<style scoped>
.notification-container {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  z-index: 1000;
}

.notification {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  animation: slide-in 0.3s ease;
}

.notification-content {
  flex: 1;
}

.notification-message {
  margin-bottom: 0.25rem;
  font-weight: bold;
}

.notification-time {
  font-size: 0.8rem;
  opacity: 0.8;
}

.notification-close {
  background: none;
  border: none;
  color: inherit;
  font-size: 1.5rem;
  cursor: pointer;
  opacity: 0.7;
  padding: 0;
  margin-left: 1rem;
}

.notification-close:hover {
  opacity: 1;
}

/* Notification types */
.notification.success {
  background-color: var(--green);
  color: var(--background);
}

.notification.error {
  background-color: var(--red);
  color: var(--foreground);
}

.notification.warning {
  background-color: var(--orange);
  color: var(--background);
}

.notification.info {
  background-color: var(--cyan);
  color: var(--background);
}

/* Animation */
.notification-enter-active,
.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from {
  transform: translateX(100%);
  opacity: 0;
}

.notification-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

@keyframes slide-in {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}
</style>