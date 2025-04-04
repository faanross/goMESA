<template>
  <div class="command-history">
    <h1>Command History</h1>

    <div class="command-list">
      <div v-if="commandHistory.length > 0">
        <div
            v-for="(cmd, index) in commandHistory"
            :key="index"
            class="command-card"
            :class="{ 'command-success': cmd.success, 'command-error': !cmd.success }"
        >
          <div class="command-header">
            <div class="command-title">
              <span class="command-text">{{ cmd.command }}</span>
              <span class="command-agent">Agent: {{ cmd.agentId }}</span>
            </div>
            <div class="command-time">
              {{ formatTime(cmd.timestamp) }}
            </div>
          </div>

          <div class="command-body">
            <div v-if="cmd.success" class="command-output">
              <pre>{{ cmd.output || '(No output)' }}</pre>
            </div>
            <div v-else class="command-error-message">
              {{ cmd.error || 'Unknown error' }}
            </div>
          </div>

          <div class="command-footer">
            <button
                v-if="isAgentActive(cmd.agentId)"
                class="primary"
                @click="rerunCommand(cmd)"
            >
              Re-run Command
            </button>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        No command history available.
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import {
  commandHistory,
  agents,
  executeCommand
} from '../services/websocket';

// Computed properties
const activeAgentIds = computed(() => {
  return agents
      .filter(agent => agent.Status === 'ALIVE')
      .map(agent => agent.ID);
});

// Methods
function formatTime(timestamp) {
  if (!timestamp) return 'Unknown';

  // Convert to local date string
  const date = new Date(timestamp);
  return date.toLocaleString();
}

function isAgentActive(agentId) {
  return activeAgentIds.value.includes(agentId);
}

function rerunCommand(cmd) {
  if (isAgentActive(cmd.agentId)) {
    executeCommand(cmd.agentId, cmd.command);
  }
}
</script>

<style scoped>
.command-history h1 {
  margin-bottom: 2rem;
  color: var(--purple);
}

.command-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.command-card {
  background-color: var(--selection);
  border-radius: 8px;
  overflow: hidden;
  border-left: 4px solid var(--comment);
}

.command-success {
  border-left-color: var(--green);
}

.command-error {
  border-left-color: var(--red);
}

.command-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1rem;
  background-color: rgba(0, 0, 0, 0.2);
}

.command-title {
  display: flex;
  flex-direction: column;
}

.command-text {
  font-family: 'Courier New', Courier, monospace;
  font-weight: bold;
  color: var(--yellow);
  margin-bottom: 0.5rem;
  word-break: break-all;
}

.command-agent {
  font-size: 0.85rem;
  color: var(--comment);
}

.command-time {
  font-size: 0.85rem;
  color: var(--comment);
  white-space: nowrap;
  margin-left: 1rem;
}

.command-body {
  padding: 1rem;
  background-color: var(--background);
  max-height: 300px;
  overflow-y: auto;
}

.command-output pre {
  font-family: 'Courier New', Courier, monospace;
  margin: 0;
  word-break: break-word;
  white-space: pre-wrap;
}

.command-error-message {
  color: var(--red);
  font-style: italic;
}

.command-footer {
  padding: 1rem;
  display: flex;
  justify-content: flex-end;
}

.empty-state {
  padding: 3rem;
  text-align: center;
  background-color: var(--selection);
  border-radius: 8px;
  color: var(--comment);
}
</style>