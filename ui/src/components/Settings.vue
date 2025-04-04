<template>
  <div class="settings">
    <h1>Settings</h1>

    <div class="card">
      <div class="card-header">
        <h2>Server Information</h2>
      </div>

      <div class="form-group">
        <label>WebSocket Status</label>
        <div class="status-display" :class="connectionState">
          {{ connectionState }}
        </div>
      </div>

      <div class="form-group">
        <label for="ws-url">WebSocket URL</label>
        <div class="flex">
          <input
              id="ws-url"
              v-model="wsUrl"
              placeholder="ws://localhost:8080/ws"
          />
          <button
              class="primary"
              @click="reconnect"
              :disabled="connectionState === 'connecting'"
          >
            {{ connectionState === 'connecting' ? 'Connecting...' : 'Connect' }}
          </button>
        </div>
      </div>
    </div>

    <div class="card mt-4">
      <div class="card-header">
        <h2>Database Management</h2>
      </div>

      <p class="settings-warning mb-3">
        Warning: These actions affect all agents and cannot be undone.
      </p>

      <div class="action-buttons">
        <button class="warning" @click="showCleanDatabaseModal = true">
          Clean Database
        </button>
      </div>
    </div>

    <!-- Clean Database Confirmation Modal -->
    <div v-if="showCleanDatabaseModal" class="modal-overlay" @click="showCleanDatabaseModal = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Clean Database</h2>
          <button class="modal-close" @click="showCleanDatabaseModal = false">×</button>
        </div>

        <div class="warning-content">
          <p>Are you sure you want to clean the database?</p>
          <p>This will remove all agent records and command history.</p>
        </div>

        <div class="modal-footer">
          <button class="primary" @click="showCleanDatabaseModal = false">Cancel</button>
          <button class="danger" @click="cleanDatabase">Confirm</button>
        </div>
      </div>
    </div>

    <!-- Shutdown Confirmation Modal -->
    <div v-if="showShutdownModal" class="modal-overlay" @click="showShutdownModal = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Shutdown Server</h2>
          <button class="modal-close" @click="showShutdownModal = false">×</button>
        </div>

        <div class="warning-content">
          <p>Are you sure you want to shutdown the server?</p>
          <p>This will kill all agents and clean the database.</p>
        </div>

        <div class="modal-footer">
          <button class="primary" @click="showShutdownModal = false">Cancel</button>
          <button class="danger" @click="shutdownServer">Confirm Shutdown</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import {
  connectWebSocket,
  connectionState,
  sendMessage,
  addNotification
} from '../services/websocket';

// State
const wsUrl = ref('ws://localhost:8080/ws');
const showCleanDatabaseModal = ref(false);
const showShutdownModal = ref(false);

// Methods
function reconnect() {
  connectWebSocket(wsUrl.value);
}

function cleanDatabase() {
  sendMessage({
    type: 'cleanDatabase'
  });

  showCleanDatabaseModal.value = false;
  addNotification('Database clean command sent', 'warning');
}

function shutdownServer() {
  sendMessage({
    type: 'shutdown'
  });

  showShutdownModal.value = false;
  addNotification('Server shutdown command sent', 'warning');
}
</script>

<style scoped>
.settings h1 {
  margin-bottom: 2rem;
  color: var(--purple);
}

.status-display {
  padding: 0.5rem 1rem;
  border-radius: 4px;
  font-weight: bold;
  display: inline-block;
}

.status-display.connected {
  background-color: var(--green);
  color: var(--background);
}

.status-display.connecting {
  background-color: var(--yellow);
  color: var(--background);
}

.status-display.disconnected {
  background-color: var(--red);
  color: var(--foreground);
}

.settings-warning {
  color: var(--red);
  font-weight: bold;
}

.action-buttons {
  display: flex;
  gap: 1rem;
}

.warning-content {
  padding: 1.5rem;
  color: var(--red);
  text-align: center;
}

.warning-content p {
  margin-bottom: 1rem;
}
</style>